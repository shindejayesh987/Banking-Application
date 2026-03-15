package service

import (
	"context"
	"errors"
	"testing"
	"time"

	intkafka "github.com/jayeshinusshinde/banking-backend/internal/kafka"
	"github.com/jayeshinusshinde/banking-backend/internal/model"
)

// --- mock repo ---

type mockAccountRepo struct {
	createFn   func(ctx context.Context, userID, accountType, currency string) (*model.Account, error)
	getByIDFn  func(ctx context.Context, id string) (*model.Account, error)
	listByUserFn func(ctx context.Context, userID string) ([]model.Account, error)
}

func (m *mockAccountRepo) Create(ctx context.Context, userID, accountType, currency string) (*model.Account, error) {
	return m.createFn(ctx, userID, accountType, currency)
}
func (m *mockAccountRepo) GetByID(ctx context.Context, id string) (*model.Account, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockAccountRepo) ListByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	return m.listByUserFn(ctx, userID)
}

// --- mock publisher ---

type mockPublisher struct {
	publishFn func(ctx context.Context, topic, key string, event intkafka.Event) error
}

func (m *mockPublisher) Publish(ctx context.Context, topic, key string, event intkafka.Event) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, topic, key, event)
	}
	return nil
}

// --- interfaces needed in account service ---

type accountRepoIface interface {
	Create(ctx context.Context, userID, accountType, currency string) (*model.Account, error)
	GetByID(ctx context.Context, id string) (*model.Account, error)
	ListByUserID(ctx context.Context, userID string) ([]model.Account, error)
}

type eventPublisherIface interface {
	Publish(ctx context.Context, topic, key string, event intkafka.Event) error
}

// accountServiceTestable lets us inject mocks for tests.
type accountServiceTestable struct {
	repo     accountRepoIface
	producer eventPublisherIface
}

func (s *accountServiceTestable) Create(ctx context.Context, req CreateAccountRequest) (*model.Account, error) {
	if req.AccountType != "savings" && req.AccountType != "checking" {
		return nil, errors.New("invalid account type: " + req.AccountType)
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	account, err := s.repo.Create(ctx, req.UserID, req.AccountType, req.Currency)
	if err != nil {
		return nil, err
	}
	go func() {
		_ = s.producer.Publish(context.Background(), intkafka.TopicAccountEvents, account.ID, intkafka.Event{
			Type:    "account.created",
			Payload: account,
		})
	}()
	return account, nil
}

func (s *accountServiceTestable) GetByID(ctx context.Context, id string) (*model.Account, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("account not found")
	}
	return account, nil
}

func (s *accountServiceTestable) ListByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	return s.repo.ListByUserID(ctx, userID)
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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func newTestService(repo accountRepoIface) *accountServiceTestable {
	return &accountServiceTestable{repo: repo, producer: &mockPublisher{}}
}

// --- Tests ---

func TestAccountCreate_Success(t *testing.T) {
	repo := &mockAccountRepo{
		createFn: func(_ context.Context, userID, accountType, currency string) (*model.Account, error) {
			a := fakeAccount()
			a.UserID = userID
			a.AccountType = accountType
			a.Currency = currency
			return a, nil
		},
	}
	svc := newTestService(repo)

	acc, err := svc.Create(context.Background(), CreateAccountRequest{
		UserID:      "usr-1",
		AccountType: "savings",
		Currency:    "USD",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acc.UserID != "usr-1" {
		t.Errorf("UserID: got %q, want usr-1", acc.UserID)
	}
}

func TestAccountCreate_InvalidType(t *testing.T) {
	svc := newTestService(&mockAccountRepo{})

	_, err := svc.Create(context.Background(), CreateAccountRequest{
		UserID:      "usr-1",
		AccountType: "investment",
	})

	if err == nil {
		t.Fatal("expected error for invalid account type, got nil")
	}
}

func TestAccountCreate_ValidTypes(t *testing.T) {
	for _, accountType := range []string{"savings", "checking"} {
		repo := &mockAccountRepo{
			createFn: func(_ context.Context, _, at, _ string) (*model.Account, error) {
				a := fakeAccount()
				a.AccountType = at
				return a, nil
			},
		}
		svc := newTestService(repo)
		acc, err := svc.Create(context.Background(), CreateAccountRequest{
			UserID:      "usr-1",
			AccountType: accountType,
		})
		if err != nil {
			t.Errorf("accountType %q: unexpected error: %v", accountType, err)
		}
		if acc.AccountType != accountType {
			t.Errorf("accountType %q: got %q", accountType, acc.AccountType)
		}
	}
}

func TestAccountCreate_DefaultCurrency(t *testing.T) {
	var capturedCurrency string
	repo := &mockAccountRepo{
		createFn: func(_ context.Context, _, _, currency string) (*model.Account, error) {
			capturedCurrency = currency
			return fakeAccount(), nil
		},
	}
	svc := newTestService(repo)

	svc.Create(context.Background(), CreateAccountRequest{
		UserID:      "usr-1",
		AccountType: "savings",
		Currency:    "", // empty → should default to USD
	})

	if capturedCurrency != "USD" {
		t.Errorf("default currency: got %q, want USD", capturedCurrency)
	}
}

func TestAccountCreate_ExplicitCurrency(t *testing.T) {
	var capturedCurrency string
	repo := &mockAccountRepo{
		createFn: func(_ context.Context, _, _, currency string) (*model.Account, error) {
			capturedCurrency = currency
			return fakeAccount(), nil
		},
	}
	svc := newTestService(repo)

	svc.Create(context.Background(), CreateAccountRequest{
		UserID:      "usr-1",
		AccountType: "savings",
		Currency:    "EUR",
	})

	if capturedCurrency != "EUR" {
		t.Errorf("explicit currency: got %q, want EUR", capturedCurrency)
	}
}

func TestAccountCreate_RepoError(t *testing.T) {
	repo := &mockAccountRepo{
		createFn: func(_ context.Context, _, _, _ string) (*model.Account, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.Create(context.Background(), CreateAccountRequest{
		UserID:      "usr-1",
		AccountType: "savings",
	})

	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

func TestAccountGetByID_Found(t *testing.T) {
	repo := &mockAccountRepo{
		getByIDFn: func(_ context.Context, id string) (*model.Account, error) {
			a := fakeAccount()
			a.ID = id
			return a, nil
		},
	}
	svc := newTestService(repo)

	acc, err := svc.GetByID(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acc.ID != "acc-1" {
		t.Errorf("ID: got %q, want acc-1", acc.ID)
	}
}

func TestAccountGetByID_NotFound(t *testing.T) {
	repo := &mockAccountRepo{
		getByIDFn: func(_ context.Context, _ string) (*model.Account, error) {
			return nil, nil // repo returns nil for not-found
		},
	}
	svc := newTestService(repo)

	_, err := svc.GetByID(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected 'account not found' error, got nil")
	}
	if err.Error() != "account not found" {
		t.Errorf("error message: got %q, want 'account not found'", err.Error())
	}
}

func TestAccountGetByID_RepoError(t *testing.T) {
	repo := &mockAccountRepo{
		getByIDFn: func(_ context.Context, _ string) (*model.Account, error) {
			return nil, errors.New("connection error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.GetByID(context.Background(), "acc-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAccountListByUserID_Success(t *testing.T) {
	repo := &mockAccountRepo{
		listByUserFn: func(_ context.Context, _ string) ([]model.Account, error) {
			return []model.Account{*fakeAccount(), *fakeAccount()}, nil
		},
	}
	svc := newTestService(repo)

	accounts, err := svc.ListByUserID(context.Background(), "usr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("count: got %d, want 2", len(accounts))
	}
}

func TestAccountListByUserID_Empty(t *testing.T) {
	repo := &mockAccountRepo{
		listByUserFn: func(_ context.Context, _ string) ([]model.Account, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo)

	accounts, err := svc.ListByUserID(context.Background(), "usr-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected empty list, got %d", len(accounts))
	}
}
