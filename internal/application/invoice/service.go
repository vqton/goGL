package invoice

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/invoice"
)

type Service interface {
	CreateInvoice(ctx context.Context, i *invoice.Invoice) error
	GetInvoice(ctx context.Context, id string) (*invoice.Invoice, error)
}

type service struct {
	repo invoice.Repository
}

func NewService(repo invoice.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateInvoice(ctx context.Context, i *invoice.Invoice) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetInvoice(ctx context.Context, id string) (*invoice.Invoice, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
