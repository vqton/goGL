package purchase

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/purchase"
)

type Service interface {
	CreateInvoice(ctx context.Context, p *purchase.PurchaseInvoice) error
	GetInvoice(ctx context.Context, id string) (*purchase.PurchaseInvoice, error)
}

type service struct {
	repo purchase.Repository
}

func NewService(repo purchase.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateInvoice(ctx context.Context, p *purchase.PurchaseInvoice) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetInvoice(ctx context.Context, id string) (*purchase.PurchaseInvoice, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
