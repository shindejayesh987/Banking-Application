package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	intkafka "github.com/jayeshinusshinde/banking-backend/internal/kafka"
	"github.com/jayeshinusshinde/banking-backend/internal/model"
	"github.com/jayeshinusshinde/banking-backend/internal/repository"
)

type TransactionService struct {
	db          *pgxpool.Pool
	accountRepo *repository.AccountRepo
	txnRepo     *repository.TransactionRepo
	producer    *intkafka.Producer
}

func NewTransactionService(
	db *pgxpool.Pool,
	accountRepo *repository.AccountRepo,
	txnRepo *repository.TransactionRepo,
	producer *intkafka.Producer,
) *TransactionService {
	return &TransactionService{
		db:          db,
		accountRepo: accountRepo,
		txnRepo:     txnRepo,
		producer:    producer,
	}
}

type DepositRequest struct {
	AccountID      string `json:"account_id"`
	Amount         int64  `json:"amount"` // cents
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

type WithdrawRequest struct {
	AccountID      string `json:"account_id"`
	Amount         int64  `json:"amount"` // cents
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

type TransferRequest struct {
	FromAccountID  string `json:"from_account_id"`
	ToAccountID    string `json:"to_account_id"`
	Amount         int64  `json:"amount"` // cents
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (s *TransactionService) Deposit(ctx context.Context, req DepositRequest) (*model.Account, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock account row
	account, err := s.accountRepo.GetByIDForUpdate(ctx, tx, req.AccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	// Create transaction record
	txnRecord := &model.Transaction{
		ToAccountID:     &req.AccountID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		TransactionType: "deposit",
		Status:          "pending",
		IdempotencyKey:  req.IdempotencyKey,
	}
	if err := s.txnRepo.Create(ctx, tx, txnRecord); err != nil {
		return nil, err
	}

	// Update balance with optimistic lock
	updated, err := s.accountRepo.UpdateBalanceOptimistic(ctx, tx, req.AccountID, req.Amount, account.Version)
	if err != nil {
		return nil, err
	}

	// Mark transaction completed
	if err := s.txnRepo.UpdateStatus(ctx, tx, req.IdempotencyKey, "completed"); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Publish event
	go func() {
		_ = s.producer.Publish(context.Background(), intkafka.TopicTransactions, req.AccountID, intkafka.Event{
			Type:    "transaction.deposit",
			Payload: map[string]interface{}{"account_id": req.AccountID, "amount": req.Amount},
		})
	}()

	return updated, nil
}

func (s *TransactionService) Withdraw(ctx context.Context, req WithdrawRequest) (*model.Account, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	account, err := s.accountRepo.GetByIDForUpdate(ctx, tx, req.AccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}
	if account.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient funds: have %d, need %d", account.Balance, req.Amount)
	}

	txnRecord := &model.Transaction{
		FromAccountID:   &req.AccountID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		TransactionType: "withdrawal",
		Status:          "pending",
		IdempotencyKey:  req.IdempotencyKey,
	}
	if err := s.txnRepo.Create(ctx, tx, txnRecord); err != nil {
		return nil, err
	}

	updated, err := s.accountRepo.UpdateBalanceOptimistic(ctx, tx, req.AccountID, -req.Amount, account.Version)
	if err != nil {
		return nil, err
	}

	if err := s.txnRepo.UpdateStatus(ctx, tx, req.IdempotencyKey, "completed"); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	go func() {
		_ = s.producer.Publish(context.Background(), intkafka.TopicTransactions, req.AccountID, intkafka.Event{
			Type:    "transaction.withdrawal",
			Payload: map[string]interface{}{"account_id": req.AccountID, "amount": req.Amount},
		})
	}()

	return updated, nil
}

func (s *TransactionService) Transfer(ctx context.Context, req TransferRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if req.FromAccountID == req.ToAccountID {
		return fmt.Errorf("cannot transfer to same account")
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock both accounts in consistent order to prevent deadlocks
	first, second := req.FromAccountID, req.ToAccountID
	if first > second {
		first, second = second, first
	}

	acct1, err := s.accountRepo.GetByIDForUpdate(ctx, tx, first)
	if err != nil || acct1 == nil {
		return fmt.Errorf("account %s not found", first)
	}
	acct2, err := s.accountRepo.GetByIDForUpdate(ctx, tx, second)
	if err != nil || acct2 == nil {
		return fmt.Errorf("account %s not found", second)
	}

	// Map back to from/to
	var fromAcct, toAcct *model.Account
	if first == req.FromAccountID {
		fromAcct, toAcct = acct1, acct2
	} else {
		fromAcct, toAcct = acct2, acct1
	}

	if fromAcct.Balance < req.Amount {
		return fmt.Errorf("insufficient funds: have %d, need %d", fromAcct.Balance, req.Amount)
	}

	txnRecord := &model.Transaction{
		FromAccountID:   &req.FromAccountID,
		ToAccountID:     &req.ToAccountID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		TransactionType: "transfer",
		Status:          "pending",
		IdempotencyKey:  req.IdempotencyKey,
	}
	if err := s.txnRepo.Create(ctx, tx, txnRecord); err != nil {
		return err
	}

	// Debit source
	if _, err := s.accountRepo.UpdateBalanceOptimistic(ctx, tx, req.FromAccountID, -req.Amount, fromAcct.Version); err != nil {
		return fmt.Errorf("debit failed: %w", err)
	}
	// Credit destination
	if _, err := s.accountRepo.UpdateBalanceOptimistic(ctx, tx, req.ToAccountID, req.Amount, toAcct.Version); err != nil {
		return fmt.Errorf("credit failed: %w", err)
	}

	if err := s.txnRepo.UpdateStatus(ctx, tx, req.IdempotencyKey, "completed"); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	go func() {
		_ = s.producer.Publish(context.Background(), intkafka.TopicTransactions, req.FromAccountID, intkafka.Event{
			Type: "transaction.transfer",
			Payload: map[string]interface{}{
				"from_account_id": req.FromAccountID,
				"to_account_id":   req.ToAccountID,
				"amount":          req.Amount,
			},
		})
	}()

	return nil
}
