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

// mockAccountService implements accountServicer for testing.
type mockAccountService struct {
	createFn      func(ctx context.Context, req service.CreateAccountRequest) (*model.Account, error)
	getByIDFn     func(ctx context.Context, id string) (*model.Account, error)
	listByUserFn  func(ctx context.Context, userID string) ([]model.Account, error)
}

func (m *mockAccountService) Create(ctx context.Context, req service.CreateAccountRequest) (*model.Account, error) {
	return m.createFn(ctx, req)
}
func (m *mockAccountService) GetByID(ctx context.Context, id string) (*model.Account, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockAccountService) ListByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	return m.listByUserFn(ctx, userID)
}

func fakeAccount() *model.Account {
	return &model.Account{
		ID:          "acc-1",
		UserID:      "usr-1",
		AccountType: "savings",
		Balance:     0,
		Currency:    "USD",
		Status:      "active",
		Version:     1,
	}
}

// --- Create ---

func TestAccountCreate_Success(t *testing.T) {
	svc := &mockAccountService{
		createFn: func(_ context.Context, _ service.CreateAccountRequest) (*model.Account, error) {
			return fakeAccount(), nil
		},
	}
	h := NewAccountHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"user_id":      "usr-1",
		"account_type": "savings",
		"currency":     "USD",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201", rr.Code)
	}
	var resp model.Account
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.ID != "acc-1" {
		t.Errorf("account ID: got %q, want acc-1", resp.ID)
	}
}

func TestAccountCreate_InvalidBody(t *testing.T) {
	h := NewAccountHandler(&mockAccountService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewBufferString("not-json"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestAccountCreate_MissingUserID(t *testing.T) {
	h := NewAccountHandler(&mockAccountService{})
	body, _ := json.Marshal(map[string]string{"account_type": "savings"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
	assertErrorContains(t, rr.Body.Bytes(), "user_id")
}

func TestAccountCreate_ServiceError(t *testing.T) {
	svc := &mockAccountService{
		createFn: func(_ context.Context, _ service.CreateAccountRequest) (*model.Account, error) {
			return nil, errors.New("invalid account type: bad")
		},
	}
	h := NewAccountHandler(svc)
	body, _ := json.Marshal(map[string]string{"user_id": "usr-1", "account_type": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", rr.Code)
	}
}

// --- GetByID ---

func TestAccountGetByID_Success(t *testing.T) {
	svc := &mockAccountService{
		getByIDFn: func(_ context.Context, id string) (*model.Account, error) {
			acc := fakeAccount()
			acc.ID = id
			return acc, nil
		},
	}
	h := NewAccountHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetByID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/acc-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

func TestAccountGetByID_NotFound(t *testing.T) {
	svc := &mockAccountService{
		getByIDFn: func(_ context.Context, _ string) (*model.Account, error) {
			return nil, errors.New("account not found")
		},
	}
	h := NewAccountHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/accounts/{id}", h.GetByID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/missing", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", rr.Code)
	}
}

// --- ListByUser ---

func TestAccountListByUser_Success(t *testing.T) {
	svc := &mockAccountService{
		listByUserFn: func(_ context.Context, _ string) ([]model.Account, error) {
			return []model.Account{*fakeAccount()}, nil
		},
	}
	h := NewAccountHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts?user_id=usr-1", nil)
	rr := httptest.NewRecorder()

	h.ListByUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var accounts []model.Account
	json.NewDecoder(rr.Body).Decode(&accounts)
	if len(accounts) != 1 {
		t.Errorf("accounts count: got %d, want 1", len(accounts))
	}
}

func TestAccountListByUser_EmptyList(t *testing.T) {
	svc := &mockAccountService{
		listByUserFn: func(_ context.Context, _ string) ([]model.Account, error) {
			return nil, nil // repo returns nil slice
		},
	}
	h := NewAccountHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts?user_id=usr-1", nil)
	rr := httptest.NewRecorder()

	h.ListByUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var accounts []model.Account
	json.NewDecoder(rr.Body).Decode(&accounts)
	if accounts == nil {
		t.Error("accounts should not be null, expected empty array []")
	}
}

func TestAccountListByUser_MissingUserID(t *testing.T) {
	h := NewAccountHandler(&mockAccountService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rr := httptest.NewRecorder()

	h.ListByUser(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestAccountCreate_ContentTypeIsJSON(t *testing.T) {
	svc := &mockAccountService{
		createFn: func(_ context.Context, _ service.CreateAccountRequest) (*model.Account, error) {
			return fakeAccount(), nil
		},
	}
	h := NewAccountHandler(svc)
	body, _ := json.Marshal(map[string]string{"user_id": "usr-1", "account_type": "checking"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
}

// helper
func assertErrorContains(t *testing.T, body []byte, substr string) {
	t.Helper()
	var resp map[string]string
	json.Unmarshal(body, &resp)
	if msg, ok := resp["error"]; !ok || len(msg) == 0 {
		t.Errorf("response missing 'error' field, body: %s", body)
	}
}
