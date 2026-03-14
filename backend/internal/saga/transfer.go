package saga

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jayeshinusshinde/banking-backend/internal/repository"
)

// TransferPayload is the data passed through all saga steps.
type TransferPayload struct {
	FromAccountID  string `json:"from_account_id"`
	ToAccountID    string `json:"to_account_id"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotency_key"`
}

// NewTransferSaga creates a saga orchestrator for account-to-account transfers.
// Steps:
//  1. Validate accounts exist and are active
//  2. Debit source account
//  3. Credit destination account
//
// Compensation:
//   - Step 2 failure → nothing to undo (debit didn't happen)
//   - Step 3 failure → re-credit the source account (undo debit)
func NewTransferSaga(db *pgxpool.Pool, accountRepo *repository.AccountRepo) *Orchestrator {
	steps := []Step{
		{
			Name: "validate_accounts",
			Execute: func(ctx context.Context, payload json.RawMessage) error {
				var p TransferPayload
				if err := json.Unmarshal(payload, &p); err != nil {
					return err
				}

				from, err := accountRepo.GetByID(ctx, p.FromAccountID)
				if err != nil || from == nil {
					return fmt.Errorf("source account not found")
				}
				if from.Status != "active" {
					return fmt.Errorf("source account is not active")
				}
				if from.Balance < p.Amount {
					return fmt.Errorf("insufficient funds: have %d, need %d", from.Balance, p.Amount)
				}

				to, err := accountRepo.GetByID(ctx, p.ToAccountID)
				if err != nil || to == nil {
					return fmt.Errorf("destination account not found")
				}
				if to.Status != "active" {
					return fmt.Errorf("destination account is not active")
				}
				return nil
			},
			Compensate: nil, // validation is read-only
		},
		{
			Name: "debit_source",
			Execute: func(ctx context.Context, payload json.RawMessage) error {
				var p TransferPayload
				if err := json.Unmarshal(payload, &p); err != nil {
					return err
				}

				tx, err := db.Begin(ctx)
				if err != nil {
					return err
				}
				defer tx.Rollback(ctx)

				acct, err := accountRepo.GetByIDForUpdate(ctx, tx, p.FromAccountID)
				if err != nil || acct == nil {
					return fmt.Errorf("source account lock failed")
				}

				_, err = accountRepo.UpdateBalanceOptimistic(ctx, tx, p.FromAccountID, -p.Amount, acct.Version)
				if err != nil {
					return fmt.Errorf("debit failed: %w", err)
				}

				return tx.Commit(ctx)
			},
			Compensate: func(ctx context.Context, payload json.RawMessage) error {
				// Undo debit: credit the source account back
				var p TransferPayload
				if err := json.Unmarshal(payload, &p); err != nil {
					return err
				}

				tx, err := db.Begin(ctx)
				if err != nil {
					return err
				}
				defer tx.Rollback(ctx)

				acct, err := accountRepo.GetByIDForUpdate(ctx, tx, p.FromAccountID)
				if err != nil || acct == nil {
					return fmt.Errorf("compensation: source account lock failed")
				}

				_, err = accountRepo.UpdateBalanceOptimistic(ctx, tx, p.FromAccountID, p.Amount, acct.Version)
				if err != nil {
					return fmt.Errorf("compensation: re-credit failed: %w", err)
				}

				return tx.Commit(ctx)
			},
		},
		{
			Name: "credit_destination",
			Execute: func(ctx context.Context, payload json.RawMessage) error {
				var p TransferPayload
				if err := json.Unmarshal(payload, &p); err != nil {
					return err
				}

				tx, err := db.Begin(ctx)
				if err != nil {
					return err
				}
				defer tx.Rollback(ctx)

				acct, err := accountRepo.GetByIDForUpdate(ctx, tx, p.ToAccountID)
				if err != nil || acct == nil {
					return fmt.Errorf("destination account lock failed")
				}

				_, err = accountRepo.UpdateBalanceOptimistic(ctx, tx, p.ToAccountID, p.Amount, acct.Version)
				if err != nil {
					return fmt.Errorf("credit failed: %w", err)
				}

				return tx.Commit(ctx)
			},
			Compensate: nil, // if credit fails, debit compensation handles rollback
		},
	}

	return NewOrchestrator(db, "transfer", steps)
}
