package costing

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/costing"
)

type Service interface {
	CreateCostSheet(ctx context.Context, c *costing.CostSheet) error
	GetCostSheet(ctx context.Context, id string) (*costing.CostSheet, error)
}

type service struct {
	repo costing.Repository
}

func NewService(repo costing.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateCostSheet(ctx context.Context, c *costing.CostSheet) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetCostSheet(ctx context.Context, id string) (*costing.CostSheet, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
