package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	jsonpkg "encoding/json"
)

// mockSagaExecutor implements sagaExecutor.
type mockSagaExecutor struct {
	executeFn func(ctx context.Context, payload jsonpkg.RawMessage) (string, error)
}

func (m *mockSagaExecutor) Execute(ctx context.Context, payload jsonpkg.RawMessage) (string, error) {
	return m.executeFn(ctx, payload)
}

// --- Saga Transfer ---

func TestSagaTransfer_Success(t *testing.T) {
	saga := &mockSagaExecutor{
		executeFn: func(_ context.Context, _ jsonpkg.RawMessage) (string, error) {
			return "saga-id-1", nil
		},
	}
	h := NewSagaTransferHandler(saga)

	body, _ := json.Marshal(map[string]interface{}{
		"from_account_id":  "acc-1",
		"to_account_id":    "acc-2",
		"amount":           500,
		"idempotency_key":  "k-saga-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "completed" {
		t.Errorf("status field: got %q, want completed", resp["status"])
	}
	if resp["saga_id"] != "saga-id-1" {
		t.Errorf("saga_id: got %q, want saga-id-1", resp["saga_id"])
	}
}

func TestSagaTransfer_InvalidBody(t *testing.T) {
	h := NewSagaTransferHandler(&mockSagaExecutor{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewBufferString("{bad"))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestSagaTransfer_MissingFromAccountID(t *testing.T) {
	h := NewSagaTransferHandler(&mockSagaExecutor{})
	body, _ := json.Marshal(map[string]interface{}{
		"to_account_id":   "acc-2",
		"amount":          100,
		"idempotency_key": "k1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestSagaTransfer_ZeroAmount(t *testing.T) {
	h := NewSagaTransferHandler(&mockSagaExecutor{})
	body, _ := json.Marshal(map[string]interface{}{
		"from_account_id": "acc-1",
		"to_account_id":   "acc-2",
		"amount":          0,
		"idempotency_key": "k2",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestSagaTransfer_SelfTransfer(t *testing.T) {
	h := NewSagaTransferHandler(&mockSagaExecutor{})
	body, _ := json.Marshal(map[string]interface{}{
		"from_account_id": "acc-1",
		"to_account_id":   "acc-1",
		"amount":          100,
		"idempotency_key": "k3",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 for self-transfer", rr.Code)
	}
	assertErrorContains(t, rr.Body.Bytes(), "same account")
}

func TestSagaTransfer_SagaError_ReturnsSagaID(t *testing.T) {
	saga := &mockSagaExecutor{
		executeFn: func(_ context.Context, _ jsonpkg.RawMessage) (string, error) {
			return "saga-id-failed", errors.New("insufficient funds")
		},
	}
	h := NewSagaTransferHandler(saga)

	body, _ := json.Marshal(map[string]interface{}{
		"from_account_id": "acc-1",
		"to_account_id":   "acc-2",
		"amount":          99999,
		"idempotency_key": "k4",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", rr.Code)
	}
	// saga_id must be returned even on failure so client can query state
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["saga_id"] != "saga-id-failed" {
		t.Errorf("saga_id in error response: got %q, want saga-id-failed", resp["saga_id"])
	}
}

func TestSagaTransfer_DefaultCurrency(t *testing.T) {
	var capturedPayload jsonpkg.RawMessage
	saga := &mockSagaExecutor{
		executeFn: func(_ context.Context, p jsonpkg.RawMessage) (string, error) {
			capturedPayload = p
			return "saga-id-2", nil
		},
	}
	h := NewSagaTransferHandler(saga)

	// No currency specified
	body, _ := json.Marshal(map[string]interface{}{
		"from_account_id": "acc-1",
		"to_account_id":   "acc-2",
		"amount":          100,
		"idempotency_key": "k5",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/saga-transfer", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	var payload map[string]interface{}
	json.Unmarshal(capturedPayload, &payload)
	if payload["currency"] != "USD" {
		t.Errorf("default currency: got %q, want USD", payload["currency"])
	}
}
