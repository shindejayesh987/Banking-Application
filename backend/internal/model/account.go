package model

import "time"

type Account struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"user_id" db:"user_id"`
	AccountType    string    `json:"account_type" db:"account_type"` // savings, checking
	Balance        int64     `json:"balance" db:"balance"`           // stored in cents
	Currency       string    `json:"currency" db:"currency"`
	Status         string    `json:"status" db:"status"` // active, frozen, closed
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	Version        int       `json:"version" db:"version"` // optimistic locking
}

type Transaction struct {
	ID              string    `json:"id" db:"id"`
	FromAccountID   *string   `json:"from_account_id" db:"from_account_id"`
	ToAccountID     *string   `json:"to_account_id" db:"to_account_id"`
	Amount          int64     `json:"amount" db:"amount"` // in cents
	Currency        string    `json:"currency" db:"currency"`
	TransactionType string    `json:"transaction_type" db:"transaction_type"` // deposit, withdrawal, transfer
	Status          string    `json:"status" db:"status"`                     // pending, completed, failed, reversed
	IdempotencyKey  string    `json:"idempotency_key" db:"idempotency_key"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	FullName  string    `json:"full_name" db:"full_name"`
	PinHash   string    `json:"-" db:"pin_hash"`
	Status    string    `json:"status" db:"status"` // active, suspended
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
