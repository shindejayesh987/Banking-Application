package service

import (
	"context"
	"fmt"

	intkafka "github.com/jayeshinusshinde/banking-backend/internal/kafka"
	"github.com/jayeshinusshinde/banking-backend/internal/model"
	"github.com/jayeshinusshinde/banking-backend/internal/repository"
)

type AccountService struct {
	repo     *repository.AccountRepo
	producer *intkafka.Producer
}

func NewAccountService(repo *repository.AccountRepo, producer *intkafka.Producer) *AccountService {
	return &AccountService{repo: repo, producer: producer}
}

type CreateAccountRequest struct {
	UserID      string `json:"user_id"`
	AccountType string `json:"account_type"`
	Currency    string `json:"currency"`
}

func (s *AccountService) Create(ctx context.Context, req CreateAccountRequest) (*model.Account, error) {
	if req.AccountType != "savings" && req.AccountType != "checking" {
		return nil, fmt.Errorf("invalid account type: %s", req.AccountType)
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	account, err := s.repo.Create(ctx, req.UserID, req.AccountType, req.Currency)
	if err != nil {
		return nil, err
	}

	// Fire-and-forget event (don't fail the request if Kafka is down)
	go func() {
		_ = s.producer.Publish(context.Background(), intkafka.TopicAccountEvents, account.ID, intkafka.Event{
			Type:    "account.created",
			Payload: account,
		})
	}()

	return account, nil
}

func (s *AccountService) GetByID(ctx context.Context, id string) (*model.Account, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}
	return account, nil
}

func (s *AccountService) ListByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	return s.repo.ListByUserID(ctx, userID)
}
