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
)

// mockUserRepository implements userRepository.
type mockUserRepository struct {
	createFn     func(ctx context.Context, email, fullName, pinHash string) (*model.User, error)
	getByIDFn    func(ctx context.Context, id string) (*model.User, error)
	getByEmailFn func(ctx context.Context, email string) (*model.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, email, fullName, pinHash string) (*model.User, error) {
	return m.createFn(ctx, email, fullName, pinHash)
}
func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func fakeUser() *model.User {
	return &model.User{
		ID:       "usr-1",
		Email:    "alice@example.com",
		FullName: "Alice Smith",
		Status:   "active",
	}
}

// --- Create ---

func TestUserCreate_Success(t *testing.T) {
	repo := &mockUserRepository{
		createFn: func(_ context.Context, _, _, _ string) (*model.User, error) {
			return fakeUser(), nil
		},
	}
	h := NewUserHandler(repo)

	body, _ := json.Marshal(map[string]string{
		"email":     "alice@example.com",
		"full_name": "Alice Smith",
		"pin":       "1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201", rr.Code)
	}
}

func TestUserCreate_InvalidBody(t *testing.T) {
	h := NewUserHandler(&mockUserRepository{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString("bad"))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestUserCreate_MissingEmail(t *testing.T) {
	h := NewUserHandler(&mockUserRepository{})
	body, _ := json.Marshal(map[string]string{"full_name": "Alice", "pin": "1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestUserCreate_MissingFullName(t *testing.T) {
	h := NewUserHandler(&mockUserRepository{})
	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "pin": "1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestUserCreate_MissingPin(t *testing.T) {
	h := NewUserHandler(&mockUserRepository{})
	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "full_name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestUserCreate_ShortPin(t *testing.T) {
	h := NewUserHandler(&mockUserRepository{})
	body, _ := json.Marshal(map[string]string{
		"email":     "a@b.com",
		"full_name": "Alice",
		"pin":       "123", // < 4 chars
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
	assertErrorContains(t, rr.Body.Bytes(), "pin")
}

func TestUserCreate_ExactlyFourCharPin(t *testing.T) {
	repo := &mockUserRepository{
		createFn: func(_ context.Context, _, _, _ string) (*model.User, error) {
			return fakeUser(), nil
		},
	}
	h := NewUserHandler(repo)
	body, _ := json.Marshal(map[string]string{
		"email":     "a@b.com",
		"full_name": "Alice",
		"pin":       "1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201 (4-char pin is valid)", rr.Code)
	}
}

func TestUserCreate_RepoError(t *testing.T) {
	repo := &mockUserRepository{
		createFn: func(_ context.Context, _, _, _ string) (*model.User, error) {
			return nil, errors.New("email already exists")
		},
	}
	h := NewUserHandler(repo)
	body, _ := json.Marshal(map[string]string{
		"email":     "dup@example.com",
		"full_name": "Alice",
		"pin":       "1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", rr.Code)
	}
}

// --- GetByID ---

func TestUserGetByID_Success(t *testing.T) {
	repo := &mockUserRepository{
		getByIDFn: func(_ context.Context, id string) (*model.User, error) {
			u := fakeUser()
			u.ID = id
			return u, nil
		},
	}
	h := NewUserHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{id}", h.GetByID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/usr-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var u model.User
	json.NewDecoder(rr.Body).Decode(&u)
	if u.ID != "usr-1" {
		t.Errorf("user ID: got %q, want usr-1", u.ID)
	}
}

func TestUserGetByID_NotFound(t *testing.T) {
	repo := &mockUserRepository{
		getByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, nil // repo returns nil for not-found
		},
	}
	h := NewUserHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{id}", h.GetByID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/ghost", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", rr.Code)
	}
}

func TestUserGetByID_RepoError(t *testing.T) {
	repo := &mockUserRepository{
		getByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewUserHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/users/{id}", h.GetByID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/usr-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}
