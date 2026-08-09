package contract

import (
	"context"

	"goGL/internal/domain/contract"
	"goGL/internal/domain/core"
)

type Service interface {
	CreateContract(ctx context.Context, c *contract.Contract) error
	GetContract(ctx context.Context, id string) (*contract.Contract, error)
	CreateLoan(ctx context.Context, l *contract.LoanAgreement) error
}

type service struct {
	repo contract.Repository
}

func NewService(repo contract.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateContract(ctx context.Context, c *contract.Contract) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetContract(ctx context.Context, id string) (*contract.Contract, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}

func (s *service) CreateLoan(ctx context.Context, l *contract.LoanAgreement) error {
	// TODO: implement
	return core.ErrNotImplemented
}
