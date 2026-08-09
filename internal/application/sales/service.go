package sales

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/sales"
)

type Service interface {
	CreateInvoice(ctx context.Context, s *sales.SalesInvoice) error
	GetInvoice(ctx context.Context, id string) (*sales.SalesInvoice, error)
}

type service struct {
	repo sales.Repository
}

func NewService(repo sales.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateInvoice(ctx context.Context, inv *sales.SalesInvoice) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetInvoice(ctx context.Context, id string) (*sales.SalesInvoice, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
