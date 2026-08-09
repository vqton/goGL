package tax

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/tax"
)

type Service interface {
	CreateDeclaration(ctx context.Context, d *tax.TaxDeclaration) error
	GetDeclaration(ctx context.Context, id string) (*tax.TaxDeclaration, error)
}

type service struct {
	repo tax.Repository
}

func NewService(repo tax.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateDeclaration(ctx context.Context, d *tax.TaxDeclaration) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetDeclaration(ctx context.Context, id string) (*tax.TaxDeclaration, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
