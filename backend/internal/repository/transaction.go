package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/jayeshinusshinde/banking-backend/internal/model"
)

type TransactionRepo struct{}

func NewTransactionRepo() *TransactionRepo {
	return &TransactionRepo{}
}

func (r *TransactionRepo) Create(ctx context.Context, tx pgx.Tx, t *model.Transaction) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO transactions (from_account_id, to_account_id, amount, currency, transaction_type, status, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.FromAccountID, t.ToAccountID, t.Amount, t.Currency, t.TransactionType, t.Status, t.IdempotencyKey,
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepo) UpdateStatus(ctx context.Context, tx pgx.Tx, idempotencyKey, status string) error {
	_, err := tx.Exec(ctx,
		`UPDATE transactions SET status = $1 WHERE idempotency_key = $2`,
		status, idempotencyKey,
	)
	if err != nil {
		return fmt.Errorf("update transaction status: %w", err)
	}
	return nil
}
