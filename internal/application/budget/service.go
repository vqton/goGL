package budget

import (
	"context"

	"goGL/internal/domain/budget"
	"goGL/internal/domain/core"
)

type Service interface {
	CreatePlan(ctx context.Context, p *budget.BudgetPlan) error
	GetPlan(ctx context.Context, id string) (*budget.BudgetPlan, error)
}

type service struct {
	repo budget.Repository
}

func NewService(repo budget.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreatePlan(ctx context.Context, p *budget.BudgetPlan) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetPlan(ctx context.Context, id string) (*budget.BudgetPlan, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
