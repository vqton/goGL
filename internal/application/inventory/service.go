package inventory

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/inventory"
)

type Service interface {
	CreateMovement(ctx context.Context, m *inventory.StockMovement) error
	GetMovement(ctx context.Context, id string) (*inventory.StockMovement, error)
}

type service struct {
	repo inventory.Repository
}

func NewService(repo inventory.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateMovement(ctx context.Context, m *inventory.StockMovement) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetMovement(ctx context.Context, id string) (*inventory.StockMovement, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
