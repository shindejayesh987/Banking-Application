package handler

import (
	"context"
	"encoding/json"

	"github.com/jayeshinusshinde/banking-backend/internal/model"
	"github.com/jayeshinusshinde/banking-backend/internal/service"
)

type accountServicer interface {
	Create(ctx context.Context, req service.CreateAccountRequest) (*model.Account, error)
	GetByID(ctx context.Context, id string) (*model.Account, error)
	ListByUserID(ctx context.Context, userID string) ([]model.Account, error)
}

type transactionServicer interface {
	Deposit(ctx context.Context, req service.DepositRequest) (*model.Account, error)
	Withdraw(ctx context.Context, req service.WithdrawRequest) (*model.Account, error)
	Transfer(ctx context.Context, req service.TransferRequest) error
}

type userRepository interface {
	Create(ctx context.Context, email, fullName, pinHash string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type sagaExecutor interface {
	Execute(ctx context.Context, payload json.RawMessage) (string, error)
}
