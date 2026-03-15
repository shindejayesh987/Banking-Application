package service

import (
	"context"
	"testing"
)

// These tests cover validation logic that fires before any DB interaction.
// Full flow tests require a running PostgreSQL instance (integration tests).

func TestDeposit_NegativeAmount(t *testing.T) {
	svc := &TransactionService{}
	_, err := svc.Deposit(context.Background(), DepositRequest{
		AccountID:      "acc-1",
		Amount:         -100,
		IdempotencyKey: "k1",
	})
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
	if err.Error() != "amount must be positive" {
		t.Errorf("error: got %q, want 'amount must be positive'", err.Error())
	}
}

func TestDeposit_ZeroAmount(t *testing.T) {
	svc := &TransactionService{}
	_, err := svc.Deposit(context.Background(), DepositRequest{
		AccountID:      "acc-1",
		Amount:         0,
		IdempotencyKey: "k2",
	})
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

func TestWithdraw_NegativeAmount(t *testing.T) {
	svc := &TransactionService{}
	_, err := svc.Withdraw(context.Background(), WithdrawRequest{
		AccountID:      "acc-1",
		Amount:         -50,
		IdempotencyKey: "k3",
	})
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
	if err.Error() != "amount must be positive" {
		t.Errorf("error: got %q, want 'amount must be positive'", err.Error())
	}
}

func TestWithdraw_ZeroAmount(t *testing.T) {
	svc := &TransactionService{}
	_, err := svc.Withdraw(context.Background(), WithdrawRequest{
		AccountID:      "acc-1",
		Amount:         0,
		IdempotencyKey: "k4",
	})
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

func TestTransfer_NegativeAmount(t *testing.T) {
	svc := &TransactionService{}
	err := svc.Transfer(context.Background(), TransferRequest{
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Amount:         -100,
		IdempotencyKey: "k5",
	})
	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
	if err.Error() != "amount must be positive" {
		t.Errorf("error: got %q, want 'amount must be positive'", err.Error())
	}
}

func TestTransfer_ZeroAmount(t *testing.T) {
	svc := &TransactionService{}
	err := svc.Transfer(context.Background(), TransferRequest{
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Amount:         0,
		IdempotencyKey: "k6",
	})
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

func TestTransfer_SameAccount(t *testing.T) {
	svc := &TransactionService{}
	err := svc.Transfer(context.Background(), TransferRequest{
		FromAccountID:  "acc-same",
		ToAccountID:    "acc-same",
		Amount:         100,
		IdempotencyKey: "k7",
	})
	if err == nil {
		t.Fatal("expected error for same-account transfer, got nil")
	}
	if err.Error() != "cannot transfer to same account" {
		t.Errorf("error: got %q, want 'cannot transfer to same account'", err.Error())
	}
}
