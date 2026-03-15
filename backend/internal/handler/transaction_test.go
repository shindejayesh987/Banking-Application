package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jayeshinusshinde/banking-backend/internal/model"
	"github.com/jayeshinusshinde/banking-backend/internal/service"
)

// mockTransactionService implements transactionServicer.
type mockTransactionService struct {
	depositFn  func(ctx context.Context, req service.DepositRequest) (*model.Account, error)
	withdrawFn func(ctx context.Context, req service.WithdrawRequest) (*model.Account, error)
	transferFn func(ctx context.Context, req service.TransferRequest) error
}

func (m *mockTransactionService) Deposit(ctx context.Context, req service.DepositRequest) (*model.Account, error) {
	return m.depositFn(ctx, req)
}
func (m *mockTransactionService) Withdraw(ctx context.Context, req service.WithdrawRequest) (*model.Account, error) {
	return m.withdrawFn(ctx, req)
}
func (m *mockTransactionService) Transfer(ctx context.Context, req service.TransferRequest) error {
	return m.transferFn(ctx, req)
}

// --- Deposit ---

func TestDeposit_Success(t *testing.T) {
	svc := &mockTransactionService{
		depositFn: func(_ context.Context, _ service.DepositRequest) (*model.Account, error) {
			acc := fakeAccount()
			acc.Balance = 1000
			return acc, nil
		},
	}
	h := NewTransactionHandler(svc)
	body, _ := json.Marshal(service.DepositRequest{
		AccountID:      "acc-1",
		Amount:         1000,
		Currency:       "USD",
		IdempotencyKey: "key-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var acc model.Account
	json.NewDecoder(rr.Body).Decode(&acc)
	if acc.Balance != 1000 {
		t.Errorf("balance: got %d, want 1000", acc.Balance)
	}
}

func TestDeposit_InvalidBody(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewBufferString("{bad"))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestDeposit_MissingAccountID(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(map[string]interface{}{"amount": 100, "idempotency_key": "k1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestDeposit_ZeroAmount(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(service.DepositRequest{
		AccountID:      "acc-1",
		Amount:         0,
		IdempotencyKey: "k1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 for zero amount", rr.Code)
	}
}

func TestDeposit_NegativeAmount(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(service.DepositRequest{
		AccountID:      "acc-1",
		Amount:         -100,
		IdempotencyKey: "k1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 for negative amount", rr.Code)
	}
}

func TestDeposit_MissingIdempotencyKey(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(map[string]interface{}{"account_id": "acc-1", "amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestDeposit_ServiceError(t *testing.T) {
	svc := &mockTransactionService{
		depositFn: func(_ context.Context, _ service.DepositRequest) (*model.Account, error) {
			return nil, errors.New("account not found")
		},
	}
	h := NewTransactionHandler(svc)
	body, _ := json.Marshal(service.DepositRequest{AccountID: "acc-x", Amount: 100, IdempotencyKey: "k1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/deposit", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Deposit(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", rr.Code)
	}
}

// --- Withdraw ---

func TestWithdraw_Success(t *testing.T) {
	svc := &mockTransactionService{
		withdrawFn: func(_ context.Context, _ service.WithdrawRequest) (*model.Account, error) {
			acc := fakeAccount()
			acc.Balance = 500
			return acc, nil
		},
	}
	h := NewTransactionHandler(svc)
	body, _ := json.Marshal(service.WithdrawRequest{AccountID: "acc-1", Amount: 500, IdempotencyKey: "k2"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/withdraw", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Withdraw(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

func TestWithdraw_MissingFields(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(map[string]interface{}{"amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/withdraw", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Withdraw(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestWithdraw_ServiceError_InsufficientFunds(t *testing.T) {
	svc := &mockTransactionService{
		withdrawFn: func(_ context.Context, _ service.WithdrawRequest) (*model.Account, error) {
			return nil, errors.New("insufficient funds: have 100, need 500")
		},
	}
	h := NewTransactionHandler(svc)
	body, _ := json.Marshal(service.WithdrawRequest{AccountID: "acc-1", Amount: 500, IdempotencyKey: "k3"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/withdraw", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Withdraw(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", rr.Code)
	}
}

// --- Transfer ---

func TestTransfer_Success(t *testing.T) {
	svc := &mockTransactionService{
		transferFn: func(_ context.Context, _ service.TransferRequest) error {
			return nil
		},
	}
	h := NewTransactionHandler(svc)
	body, _ := json.Marshal(service.TransferRequest{
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Amount:         200,
		IdempotencyKey: "k4",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "completed" {
		t.Errorf("response status: got %q, want completed", resp["status"])
	}
}

func TestTransfer_MissingFromAccountID(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(map[string]interface{}{
		"to_account_id":   "acc-2",
		"amount":          200,
		"idempotency_key": "k5",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestTransfer_ZeroAmount(t *testing.T) {
	h := NewTransactionHandler(&mockTransactionService{})
	body, _ := json.Marshal(service.TransferRequest{
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Amount:         0,
		IdempotencyKey: "k6",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestTransfer_ServiceError(t *testing.T) {
	svc := &mockTransactionService{
		transferFn: func(_ context.Context, _ service.TransferRequest) error {
			return errors.New("insufficient funds")
		},
	}
	h := NewTransactionHandler(svc)
	body, _ := json.Marshal(service.TransferRequest{
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Amount:         1000,
		IdempotencyKey: "k7",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", rr.Code)
	}
}
