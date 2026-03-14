package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jayeshinusshinde/banking-backend/internal/model"
)

type AccountRepo struct {
	db *pgxpool.Pool
}

func NewAccountRepo(db *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{db: db}
}

func (r *AccountRepo) Create(ctx context.Context, userID, accountType, currency string) (*model.Account, error) {
	var a model.Account
	err := r.db.QueryRow(ctx,
		`INSERT INTO accounts (user_id, account_type, currency)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, account_type, balance, currency, status, version, created_at, updated_at`,
		userID, accountType, currency,
	).Scan(&a.ID, &a.UserID, &a.AccountType, &a.Balance, &a.Currency, &a.Status, &a.Version, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return &a, nil
}

func (r *AccountRepo) GetByID(ctx context.Context, id string) (*model.Account, error) {
	var a model.Account
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, account_type, balance, currency, status, version, created_at, updated_at
		 FROM accounts WHERE id = $1`, id,
	).Scan(&a.ID, &a.UserID, &a.AccountType, &a.Balance, &a.Currency, &a.Status, &a.Version, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return &a, nil
}

func (r *AccountRepo) ListByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, account_type, balance, currency, status, version, created_at, updated_at
		 FROM accounts WHERE user_id = $1 ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.AccountType, &a.Balance, &a.Currency, &a.Status, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// UpdateBalanceOptimistic updates balance only if version matches (optimistic lock).
// Returns the updated account or an error if the version has changed.
func (r *AccountRepo) UpdateBalanceOptimistic(ctx context.Context, tx pgx.Tx, id string, delta int64, expectedVersion int) (*model.Account, error) {
	var a model.Account
	err := tx.QueryRow(ctx,
		`UPDATE accounts
		 SET balance = balance + $1, version = version + 1, updated_at = now()
		 WHERE id = $2 AND version = $3 AND status = 'active'
		 RETURNING id, user_id, account_type, balance, currency, status, version, created_at, updated_at`,
		delta, id, expectedVersion,
	).Scan(&a.ID, &a.UserID, &a.AccountType, &a.Balance, &a.Currency, &a.Status, &a.Version, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("optimistic lock conflict or account not active")
	}
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}
	return &a, nil
}

// GetByIDForUpdate acquires a row-level lock within a transaction.
func (r *AccountRepo) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id string) (*model.Account, error) {
	var a model.Account
	err := tx.QueryRow(ctx,
		`SELECT id, user_id, account_type, balance, currency, status, version, created_at, updated_at
		 FROM accounts WHERE id = $1 FOR UPDATE`, id,
	).Scan(&a.ID, &a.UserID, &a.AccountType, &a.Balance, &a.Currency, &a.Status, &a.Version, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get account for update: %w", err)
	}
	return &a, nil
}
