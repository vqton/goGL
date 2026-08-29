package budget

import (
	"context"
	"fmt"

	"goGL/internal/domain/budget"
	"goGL/internal/domain/core"
)

type Service interface {
	Create(ctx context.Context, p *budget.BudgetPlan, actor string) (*budget.BudgetPlan, error)
	Get(ctx context.Context, id string) (*budget.BudgetPlan, error)
	Update(ctx context.Context, id string, patch *budget.BudgetPlan, actor string) (*budget.BudgetPlan, error)
	List(ctx context.Context, fiscalYear int, department string) ([]*budget.BudgetPlan, error)
	Approve(ctx context.Context, id, actor string) (*budget.BudgetPlan, error)
	Lock(ctx context.Context, id, actor string) (*budget.BudgetPlan, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo budget.Repository
}

func NewService(repo budget.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, p *budget.BudgetPlan, actor string) (*budget.BudgetPlan, error) {
	plan := p.Clone()
	plan.State = budget.BudgetStateDraft
	plan.CreatedBy = actor
	plan.UpdatedBy = actor

	if err := budget.ValidatePlan(plan); err != nil {
		return nil, err
	}

	n, err := s.repo.NextCode(ctx, plan.FiscalYear)
	if err != nil {
		return nil, err
	}
	plan.Code = fmt.Sprintf("BP-%d-%04d", plan.FiscalYear, n)
	plan.ID = core.RowID("budget", plan.Code)

	now := core.NowRFC3339()
	plan.CreatedAt = now
	plan.UpdatedAt = now
	plan.Recalculate()

	if err := s.repo.Create(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *service) Get(ctx context.Context, id string) (*budget.BudgetPlan, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, patch *budget.BudgetPlan, actor string) (*budget.BudgetPlan, error) {
	cur, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.State != budget.BudgetStateDraft {
		return nil, budget.ErrLocked
	}

	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Department != "" {
		cur.Department = patch.Department
	}
	if patch.Period != "" {
		cur.Period = patch.Period
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	if patch.Items != nil {
		cur.Items = patch.Items
	}

	if err := budget.ValidatePlan(cur); err != nil {
		return nil, err
	}

	cur.Recalculate()
	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) List(ctx context.Context, fiscalYear int, department string) ([]*budget.BudgetPlan, error) {
	return s.repo.List(ctx, fiscalYear, department)
}

func (s *service) Approve(ctx context.Context, id, actor string) (*budget.BudgetPlan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan.State != budget.BudgetStateDraft {
		return nil, budget.ErrLocked
	}

	plan.State = budget.BudgetStateApproved
	plan.ApprovedBy = actor
	plan.ApprovedAt = core.NowRFC3339()
	plan.UpdatedBy = actor
	plan.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *service) Lock(ctx context.Context, id, actor string) (*budget.BudgetPlan, error) {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan.State != budget.BudgetStateApproved {
		return nil, budget.ErrLocked
	}

	plan.State = budget.BudgetStateLocked
	plan.UpdatedBy = actor
	plan.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	plan, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if plan.State != budget.BudgetStateDraft {
		return budget.ErrLocked
	}
	return s.repo.Delete(ctx, id)
}
