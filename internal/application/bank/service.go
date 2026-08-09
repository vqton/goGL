package bank

import (
	"context"

	"goGL/internal/domain/bank"
	"goGL/internal/domain/core"
)

type Service interface {
	CreateTransaction(ctx context.Context, t *bank.BankTransaction) error
	GetTransaction(ctx context.Context, id string) (*bank.BankTransaction, error)
}

type service struct {
	repo bank.Repository
}

func NewService(repo bank.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTransaction(ctx context.Context, t *bank.BankTransaction) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetTransaction(ctx context.Context, id string) (*bank.BankTransaction, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
