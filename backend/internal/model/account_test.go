package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAccount_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	a := Account{
		ID:          "acc-1",
		UserID:      "usr-1",
		AccountType: "savings",
		Balance:     10000,
		Currency:    "USD",
		Status:      "active",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal Account: %v", err)
	}

	var decoded Account
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal Account: %v", err)
	}

	if decoded.ID != a.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, a.ID)
	}
	if decoded.Balance != a.Balance {
		t.Errorf("Balance: got %d, want %d", decoded.Balance, a.Balance)
	}
	if decoded.AccountType != a.AccountType {
		t.Errorf("AccountType: got %q, want %q", decoded.AccountType, a.AccountType)
	}
	if decoded.Version != a.Version {
		t.Errorf("Version: got %d, want %d", decoded.Version, a.Version)
	}
}

func TestTransaction_JSONSerialization(t *testing.T) {
	fromID := "acc-from"
	toID := "acc-to"
	txn := Transaction{
		ID:              "txn-1",
		FromAccountID:   &fromID,
		ToAccountID:     &toID,
		Amount:          5000,
		Currency:        "USD",
		TransactionType: "transfer",
		Status:          "completed",
		IdempotencyKey:  "key-123",
		CreatedAt:       time.Now(),
	}

	data, err := json.Marshal(txn)
	if err != nil {
		t.Fatalf("marshal Transaction: %v", err)
	}

	var decoded Transaction
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal Transaction: %v", err)
	}

	if decoded.ID != txn.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, txn.ID)
	}
	if decoded.Amount != txn.Amount {
		t.Errorf("Amount: got %d, want %d", decoded.Amount, txn.Amount)
	}
	if decoded.IdempotencyKey != txn.IdempotencyKey {
		t.Errorf("IdempotencyKey: got %q, want %q", decoded.IdempotencyKey, txn.IdempotencyKey)
	}
	if decoded.FromAccountID == nil || *decoded.FromAccountID != fromID {
		t.Errorf("FromAccountID: got %v, want %q", decoded.FromAccountID, fromID)
	}
	if decoded.ToAccountID == nil || *decoded.ToAccountID != toID {
		t.Errorf("ToAccountID: got %v, want %q", decoded.ToAccountID, toID)
	}
}

func TestTransaction_NilAccountIDs(t *testing.T) {
	txn := Transaction{
		ID:              "txn-deposit",
		FromAccountID:   nil,
		ToAccountID:     nil,
		Amount:          1000,
		Currency:        "USD",
		TransactionType: "deposit",
		Status:          "pending",
	}

	data, err := json.Marshal(txn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Transaction
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.FromAccountID != nil {
		t.Errorf("FromAccountID should be nil, got %v", decoded.FromAccountID)
	}
}

func TestUser_JSONSerialization_PinHashOmitted(t *testing.T) {
	u := User{
		ID:        "usr-1",
		Email:     "test@example.com",
		FullName:  "Test User",
		PinHash:   "super-secret-hash",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal User: %v", err)
	}

	// PinHash has json:"-" tag — must not appear in JSON
	if strings.Contains(string(data), "super-secret-hash") {
		t.Error("PinHash should not appear in JSON output")
	}
	if strings.Contains(string(data), "pin_hash") {
		t.Error("pin_hash key should not appear in JSON output")
	}

	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal User: %v", err)
	}

	if decoded.Email != u.Email {
		t.Errorf("Email: got %q, want %q", decoded.Email, u.Email)
	}
	if decoded.PinHash != "" {
		t.Errorf("PinHash should be empty after decode, got %q", decoded.PinHash)
	}
}

func TestUser_StatusField(t *testing.T) {
	u := User{Status: "active"}
	data, _ := json.Marshal(u)
	if !strings.Contains(string(data), `"status":"active"`) {
		t.Errorf("status field missing from JSON: %s", data)
	}
}
