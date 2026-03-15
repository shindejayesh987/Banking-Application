//go:build integration

// Package integration contains end-to-end tests that run against a real
// PostgreSQL instance.  They require the INTEGRATION_DSN environment variable
// (or fall back to the default below) and are opt-in via the "integration" build
// tag:
//
//	go test -tags integration ./integration/... -v
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/jayeshinusshinde/banking-backend/internal/handler"
	intkafka "github.com/jayeshinusshinde/banking-backend/internal/kafka"
	"github.com/jayeshinusshinde/banking-backend/internal/middleware"
	"github.com/jayeshinusshinde/banking-backend/internal/repository"
	"github.com/jayeshinusshinde/banking-backend/internal/saga"
	"github.com/jayeshinusshinde/banking-backend/internal/service"
)

// ──────────────────────────────────────────────────────────────────────────────
// Test harness
// ──────────────────────────────────────────────────────────────────────────────

const defaultDSN = "postgres://banking:banking123@localhost:5432/banking_test"

// newTestServer wires up the full HTTP router against a real Postgres pool and
// a miniredis instance, returning an *httptest.Server ready to accept requests.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()

	// ── PostgreSQL ────────────────────────────────────────────────────────────
	dsn := os.Getenv("INTEGRATION_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("postgres ping failed (%s): %v — is the container running?", dsn, err)
	}

	// Apply schema once per test run (idempotent due to IF NOT EXISTS).
	applySchema(t, ctx, pool)

	t.Cleanup(func() {
		cleanSchema(t, pool)
		pool.Close()
	})

	// ── miniredis (for rate limiter) ─────────────────────────────────────────
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	// ── no-op Kafka writer ───────────────────────────────────────────────────
	// Point at a non-existent broker; writes fail silently (async). The
	// integration tests do not assert on Kafka events.
	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP("localhost:9999"),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		MaxAttempts:  1,
	}
	t.Cleanup(func() { kafkaWriter.Close() })

	// ── Dependency graph ─────────────────────────────────────────────────────
	producer := intkafka.NewProducer(kafkaWriter)

	accountRepo := repository.NewAccountRepo(pool)
	txnRepo := repository.NewTransactionRepo()
	userRepo := repository.NewUserRepo(pool)

	accountSvc := service.NewAccountService(accountRepo, producer)
	txnSvc := service.NewTransactionService(pool, accountRepo, txnRepo, producer)
	transferSaga := saga.NewTransferSaga(pool, accountRepo)

	userH := handler.NewUserHandler(userRepo)
	accountH := handler.NewAccountHandler(accountSvc)
	txnH := handler.NewTransactionHandler(txnSvc)
	sagaH := handler.NewSagaTransferHandler(transferSaga)

	// ── Router ───────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/users", userH.Create)
	mux.HandleFunc("GET /api/v1/users/{id}", userH.GetByID)
	mux.HandleFunc("POST /api/v1/accounts", accountH.Create)
	mux.HandleFunc("GET /api/v1/accounts/{id}", accountH.GetByID)
	mux.HandleFunc("GET /api/v1/accounts", accountH.ListByUser)
	mux.HandleFunc("POST /api/v1/transactions/deposit", txnH.Deposit)
	mux.HandleFunc("POST /api/v1/transactions/withdraw", txnH.Withdraw)
	mux.HandleFunc("POST /api/v1/transactions/transfer", txnH.Transfer)
	mux.HandleFunc("POST /api/v1/transactions/saga-transfer", sagaH.Transfer)

	rateLimiter := middleware.NewRateLimiter(rdb, 1000, time.Minute)
	return httptest.NewServer(rateLimiter.Middleware(mux))
}

// applySchema creates all tables using the production migration SQL.
func applySchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	sql := `
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";

		CREATE TABLE IF NOT EXISTS users (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email      VARCHAR(255) NOT NULL UNIQUE,
			full_name  VARCHAR(255) NOT NULL,
			pin_hash   VARCHAR(255) NOT NULL,
			status     VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
			created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS accounts (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id      UUID         NOT NULL REFERENCES users(id),
			account_type VARCHAR(20)  NOT NULL CHECK (account_type IN ('savings','checking')),
			balance      BIGINT       NOT NULL DEFAULT 0 CHECK (balance >= 0),
			currency     VARCHAR(3)   NOT NULL DEFAULT 'USD',
			status       VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active','frozen','closed')),
			version      INT          NOT NULL DEFAULT 1,
			created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
			updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS transactions (
			id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			from_account_id  UUID REFERENCES accounts(id),
			to_account_id    UUID REFERENCES accounts(id),
			amount           BIGINT       NOT NULL CHECK (amount > 0),
			currency         VARCHAR(3)   NOT NULL DEFAULT 'USD',
			transaction_type VARCHAR(20)  NOT NULL CHECK (transaction_type IN ('deposit','withdrawal','transfer')),
			status           VARCHAR(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','completed','failed','reversed')),
			idempotency_key  VARCHAR(255) NOT NULL UNIQUE,
			created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
			CONSTRAINT chk_transaction_accounts CHECK (
				(transaction_type = 'deposit'    AND to_account_id IS NOT NULL) OR
				(transaction_type = 'withdrawal' AND from_account_id IS NOT NULL) OR
				(transaction_type = 'transfer'   AND from_account_id IS NOT NULL AND to_account_id IS NOT NULL)
			)
		);

		CREATE TABLE IF NOT EXISTS saga_state (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			saga_type    VARCHAR(50)  NOT NULL,
			status       VARCHAR(20)  NOT NULL DEFAULT 'started' CHECK (status IN ('started','completed','compensating','failed')),
			payload      JSONB        NOT NULL DEFAULT '{}',
			current_step INT          NOT NULL DEFAULT 0,
			created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
			updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
		);
	`
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("applySchema: %v", err)
	}
}

// cleanSchema truncates all test data so each test starts with a clean slate.
func cleanSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`TRUNCATE saga_state, transactions, accounts, users RESTART IDENTITY CASCADE`,
	); err != nil {
		t.Logf("cleanSchema truncate: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func postJSON(t *testing.T, srv *httptest.Server, path string, body interface{}) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := srv.Client().Post(srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func getJSON(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := srv.Client().Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

// ──────────────────────────────────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestFullFlow_CreateUser_CreateAccount_Deposit_Withdraw_Transfer is the
// primary end-to-end flow test.  It exercises every API endpoint in a single
// realistic scenario:
//
//  1. Create two users
//  2. Each user opens a checking account
//  3. Deposit funds into account A
//  4. Withdraw a portion from account A
//  5. Transfer the remainder from A → B
//  6. Assert DB balances are correct
func TestFullFlow_CreateUser_CreateAccount_Deposit_Withdraw_Transfer(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	// ── Step 1: Create users ─────────────────────────────────────────────────
	respA := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email": fmt.Sprintf("alice+%d@example.com", time.Now().UnixNano()),
		"full_name": "Alice",
		"pin":       "1234",
	})
	if respA.StatusCode != http.StatusCreated {
		body := decodeBody(t, respA)
		t.Fatalf("create user A: want 201, got %d — %v", respA.StatusCode, body)
	}
	userA := decodeBody(t, respA)
	userAID := userA["id"].(string)

	respB := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email":     fmt.Sprintf("bob+%d@example.com", time.Now().UnixNano()),
		"full_name": "Bob",
		"pin":       "5678",
	})
	if respB.StatusCode != http.StatusCreated {
		t.Fatalf("create user B: want 201, got %d", respB.StatusCode)
	}
	userB := decodeBody(t, respB)
	userBID := userB["id"].(string)

	// ── Step 2: Create accounts ──────────────────────────────────────────────
	accARsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{
		"user_id":      userAID,
		"account_type": "checking",
	})
	if accARsp.StatusCode != http.StatusCreated {
		body := decodeBody(t, accARsp)
		t.Fatalf("create account A: want 201, got %d — %v", accARsp.StatusCode, body)
	}
	accA := decodeBody(t, accARsp)
	accAID := accA["id"].(string)

	accBRsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{
		"user_id":      userBID,
		"account_type": "savings",
	})
	if accBRsp.StatusCode != http.StatusCreated {
		t.Fatalf("create account B: want 201, got %d", accBRsp.StatusCode)
	}
	accB := decodeBody(t, accBRsp)
	accBID := accB["id"].(string)

	// ── Step 3: Deposit 10 000 cents ($100) into account A ───────────────────
	depRsp := postJSON(t, srv, "/api/v1/transactions/deposit", map[string]interface{}{
		"account_id":      accAID,
		"amount":          10000,
		"idempotency_key": fmt.Sprintf("dep-a-%d", time.Now().UnixNano()),
	})
	if depRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, depRsp)
		t.Fatalf("deposit: want 200, got %d — %v", depRsp.StatusCode, body)
	}
	depResult := decodeBody(t, depRsp)
	if int64(depResult["balance"].(float64)) != 10000 {
		t.Errorf("after deposit: want balance 10000, got %v", depResult["balance"])
	}

	// ── Step 4: Withdraw 3 000 cents ($30) from account A ────────────────────
	wdRsp := postJSON(t, srv, "/api/v1/transactions/withdraw", map[string]interface{}{
		"account_id":      accAID,
		"amount":          3000,
		"idempotency_key": fmt.Sprintf("wd-a-%d", time.Now().UnixNano()),
	})
	if wdRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, wdRsp)
		t.Fatalf("withdraw: want 200, got %d — %v", wdRsp.StatusCode, body)
	}
	wdResult := decodeBody(t, wdRsp)
	if int64(wdResult["balance"].(float64)) != 7000 {
		t.Errorf("after withdraw: want balance 7000, got %v", wdResult["balance"])
	}

	// ── Step 5: Transfer 5 000 cents ($50) from A → B ────────────────────────
	trRsp := postJSON(t, srv, "/api/v1/transactions/transfer", map[string]interface{}{
		"from_account_id": accAID,
		"to_account_id":   accBID,
		"amount":          5000,
		"idempotency_key": fmt.Sprintf("tr-ab-%d", time.Now().UnixNano()),
	})
	if trRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, trRsp)
		t.Fatalf("transfer: want 200, got %d — %v", trRsp.StatusCode, body)
	}
	trResult := decodeBody(t, trRsp)
	if trResult["status"] != "completed" {
		t.Errorf("transfer status: want completed, got %v", trResult["status"])
	}

	// ── Step 6: Verify final balances via GET /accounts/{id} ────────────────
	getA := getJSON(t, srv, "/api/v1/accounts/"+accAID)
	if getA.StatusCode != http.StatusOK {
		t.Fatalf("get account A: want 200, got %d", getA.StatusCode)
	}
	finalA := decodeBody(t, getA)
	if int64(finalA["balance"].(float64)) != 2000 {
		t.Errorf("final balance A: want 2000 (10000-3000-5000), got %v", finalA["balance"])
	}

	getB := getJSON(t, srv, "/api/v1/accounts/"+accBID)
	if getB.StatusCode != http.StatusOK {
		t.Fatalf("get account B: want 200, got %d", getB.StatusCode)
	}
	finalB := decodeBody(t, getB)
	if int64(finalB["balance"].(float64)) != 5000 {
		t.Errorf("final balance B: want 5000, got %v", finalB["balance"])
	}
}

// TestSagaTransfer_FullFlow tests the saga-based transfer path independently.
func TestSagaTransfer_FullFlow(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	// Create a user and two accounts
	userRsp := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email":     fmt.Sprintf("saga+%d@example.com", time.Now().UnixNano()),
		"full_name": "Saga Tester",
		"pin":       "0000",
	})
	if userRsp.StatusCode != http.StatusCreated {
		t.Fatalf("create user: %d", userRsp.StatusCode)
	}
	user := decodeBody(t, userRsp)
	uid := user["id"].(string)

	acc1Rsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{"user_id": uid, "account_type": "checking"})
	if acc1Rsp.StatusCode != http.StatusCreated {
		t.Fatalf("create acc1: %d", acc1Rsp.StatusCode)
	}
	acc1 := decodeBody(t, acc1Rsp)
	acc1ID := acc1["id"].(string)

	acc2Rsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{"user_id": uid, "account_type": "savings"})
	if acc2Rsp.StatusCode != http.StatusCreated {
		t.Fatalf("create acc2: %d", acc2Rsp.StatusCode)
	}
	acc2 := decodeBody(t, acc2Rsp)
	acc2ID := acc2["id"].(string)

	// Fund account 1
	depRsp := postJSON(t, srv, "/api/v1/transactions/deposit", map[string]interface{}{
		"account_id":      acc1ID,
		"amount":          20000,
		"idempotency_key": fmt.Sprintf("saga-dep-%d", time.Now().UnixNano()),
	})
	if depRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, depRsp)
		t.Fatalf("deposit for saga: %d — %v", depRsp.StatusCode, body)
	}
	decodeBody(t, depRsp)

	// Saga transfer 8 000 cents from acc1 → acc2
	sagaRsp := postJSON(t, srv, "/api/v1/transactions/saga-transfer", map[string]interface{}{
		"from_account_id": acc1ID,
		"to_account_id":   acc2ID,
		"amount":          8000,
		"currency":        "USD",
		"idempotency_key": fmt.Sprintf("saga-tr-%d", time.Now().UnixNano()),
	})
	if sagaRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, sagaRsp)
		t.Fatalf("saga transfer: want 200, got %d — %v", sagaRsp.StatusCode, body)
	}
	sagaResult := decodeBody(t, sagaRsp)
	if sagaResult["status"] != "completed" {
		t.Errorf("saga status: want completed, got %v", sagaResult["status"])
	}
	if sagaResult["saga_id"] == "" {
		t.Error("saga_id should be present in response")
	}

	// Verify balances
	getAcc1 := getJSON(t, srv, "/api/v1/accounts/"+acc1ID)
	final1 := decodeBody(t, getAcc1)
	if int64(final1["balance"].(float64)) != 12000 {
		t.Errorf("saga: acc1 balance want 12000, got %v", final1["balance"])
	}

	getAcc2 := getJSON(t, srv, "/api/v1/accounts/"+acc2ID)
	final2 := decodeBody(t, getAcc2)
	if int64(final2["balance"].(float64)) != 8000 {
		t.Errorf("saga: acc2 balance want 8000, got %v", final2["balance"])
	}
}

// TestWithdraw_InsufficientFunds verifies the API returns 422 when the account
// has insufficient funds.
func TestWithdraw_InsufficientFunds(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	userRsp := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email":     fmt.Sprintf("broke+%d@example.com", time.Now().UnixNano()),
		"full_name": "Broke User",
		"pin":       "1234",
	})
	user := decodeBody(t, userRsp)
	uid := user["id"].(string)

	accRsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{"user_id": uid, "account_type": "checking"})
	acc := decodeBody(t, accRsp)
	accID := acc["id"].(string)

	// Deposit 100 cents
	depRsp := postJSON(t, srv, "/api/v1/transactions/deposit", map[string]interface{}{
		"account_id":      accID,
		"amount":          100,
		"idempotency_key": fmt.Sprintf("small-dep-%d", time.Now().UnixNano()),
	})
	if depRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, depRsp)
		t.Fatalf("deposit: %d — %v", depRsp.StatusCode, body)
	}
	decodeBody(t, depRsp)

	// Try to withdraw more than balance
	wdRsp := postJSON(t, srv, "/api/v1/transactions/withdraw", map[string]interface{}{
		"account_id":      accID,
		"amount":          9999,
		"idempotency_key": fmt.Sprintf("overdraft-%d", time.Now().UnixNano()),
	})
	if wdRsp.StatusCode != http.StatusUnprocessableEntity {
		body := decodeBody(t, wdRsp)
		t.Fatalf("overdraft: want 422, got %d — %v", wdRsp.StatusCode, body)
	}
	body := decodeBody(t, wdRsp)
	if body["error"] == nil {
		t.Error("expected error field in response")
	}
}

// TestIdempotency verifies that replaying the same deposit with the same
// idempotency key does not double-credit the account.
func TestIdempotency_Deposit(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	userRsp := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email":     fmt.Sprintf("idem+%d@example.com", time.Now().UnixNano()),
		"full_name": "Idempotency User",
		"pin":       "1234",
	})
	user := decodeBody(t, userRsp)
	uid := user["id"].(string)

	accRsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{"user_id": uid, "account_type": "checking"})
	acc := decodeBody(t, accRsp)
	accID := acc["id"].(string)

	key := fmt.Sprintf("idem-dep-%d", time.Now().UnixNano())

	// First deposit — should succeed
	r1 := postJSON(t, srv, "/api/v1/transactions/deposit", map[string]interface{}{
		"account_id":      accID,
		"amount":          5000,
		"idempotency_key": key,
	})
	if r1.StatusCode != http.StatusOK {
		body := decodeBody(t, r1)
		t.Fatalf("first deposit: want 200, got %d — %v", r1.StatusCode, body)
	}
	decodeBody(t, r1)

	// Second deposit with same key — should return 422 (duplicate idempotency key)
	r2 := postJSON(t, srv, "/api/v1/transactions/deposit", map[string]interface{}{
		"account_id":      accID,
		"amount":          5000,
		"idempotency_key": key,
	})
	if r2.StatusCode != http.StatusUnprocessableEntity {
		body := decodeBody(t, r2)
		t.Fatalf("duplicate deposit: want 422, got %d — %v", r2.StatusCode, body)
	}
	decodeBody(t, r2)

	// Balance should still be 5 000, not 10 000
	getAcc := getJSON(t, srv, "/api/v1/accounts/"+accID)
	finalAcc := decodeBody(t, getAcc)
	if int64(finalAcc["balance"].(float64)) != 5000 {
		t.Errorf("idempotency: want balance 5000, got %v", finalAcc["balance"])
	}
}

// TestGetUser_NotFound verifies 404 for a non-existent user ID.
func TestGetUser_NotFound(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := getJSON(t, srv, "/api/v1/users/00000000-0000-0000-0000-000000000000")
	if resp.StatusCode != http.StatusNotFound {
		body := decodeBody(t, resp)
		t.Fatalf("want 404 for unknown user, got %d — %v", resp.StatusCode, body)
	}
	decodeBody(t, resp)
}

// TestListAccounts verifies that all accounts for a user are returned.
func TestListAccounts(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	userRsp := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email":     fmt.Sprintf("list+%d@example.com", time.Now().UnixNano()),
		"full_name": "List Tester",
		"pin":       "1234",
	})
	user := decodeBody(t, userRsp)
	uid := user["id"].(string)

	// Create 3 accounts
	for i, accType := range []string{"checking", "savings", "checking"} {
		r := postJSON(t, srv, "/api/v1/accounts", map[string]string{
			"user_id":      uid,
			"account_type": accType,
		})
		if r.StatusCode != http.StatusCreated {
			body := decodeBody(t, r)
			t.Fatalf("create account %d: want 201, got %d — %v", i, r.StatusCode, body)
		}
		decodeBody(t, r)
	}

	// List accounts
	listRsp := getJSON(t, srv, "/api/v1/accounts?user_id="+uid)
	if listRsp.StatusCode != http.StatusOK {
		body := decodeBody(t, listRsp)
		t.Fatalf("list accounts: want 200, got %d — %v", listRsp.StatusCode, body)
	}
	defer listRsp.Body.Close()

	var accounts []interface{}
	if err := json.NewDecoder(listRsp.Body).Decode(&accounts); err != nil {
		t.Fatalf("decode account list: %v", err)
	}
	if len(accounts) != 3 {
		t.Errorf("want 3 accounts, got %d", len(accounts))
	}
}

// TestSelfTransfer_Rejected verifies that transferring to the same account
// returns 400.
func TestSelfTransfer_Rejected(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	userRsp := postJSON(t, srv, "/api/v1/users", map[string]string{
		"email":     fmt.Sprintf("self+%d@example.com", time.Now().UnixNano()),
		"full_name": "Self Transferer",
		"pin":       "1234",
	})
	user := decodeBody(t, userRsp)
	uid := user["id"].(string)

	accRsp := postJSON(t, srv, "/api/v1/accounts", map[string]string{"user_id": uid, "account_type": "checking"})
	acc := decodeBody(t, accRsp)
	accID := acc["id"].(string)

	resp := postJSON(t, srv, "/api/v1/transactions/saga-transfer", map[string]interface{}{
		"from_account_id": accID,
		"to_account_id":   accID,
		"amount":          100,
		"idempotency_key": fmt.Sprintf("self-%d", time.Now().UnixNano()),
	})
	if resp.StatusCode != http.StatusBadRequest {
		body := decodeBody(t, resp)
		t.Fatalf("self transfer: want 400, got %d — %v", resp.StatusCode, body)
	}
	decodeBody(t, resp)
}
