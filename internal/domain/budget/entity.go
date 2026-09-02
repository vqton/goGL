package budget

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound  = errors.New("budget: not found")
	ErrDuplicate = errors.New("budget: duplicate")
	ErrInvalid   = errors.New("budget: invalid input")
	ErrLocked    = errors.New("budget: plan is locked")
)

type BudgetState string

const (
	BudgetStateDraft    BudgetState = "draft"
	BudgetStateApproved BudgetState = "approved"
	BudgetStateLocked   BudgetState = "locked"
)

type BudgetItem struct {
	CategoryCode string `json:"category_code"`
	Planned      int64  `json:"planned"`
	Actual       int64  `json:"actual"`
	Description  string `json:"description,omitempty"`
}

type BudgetPlan struct {
	ID           string       `json:"id"`
	Code         string       `json:"code"`
	Name         string       `json:"name"`
	Department   string       `json:"department,omitempty"`
	FiscalYear   int          `json:"fiscal_year"`
	Period       string       `json:"period"`
	Items        []BudgetItem `json:"items"`
	State        BudgetState  `json:"state"`
	TotalPlanned int64        `json:"total_planned"`
	TotalActual  int64        `json:"total_actual"`
	Notes        string       `json:"notes,omitempty"`
	CreatedBy    string       `json:"created_by,omitempty"`
	CreatedAt    string       `json:"created_at"`
	UpdatedBy    string       `json:"updated_by,omitempty"`
	UpdatedAt    string       `json:"updated_at"`
	ApprovedBy   string       `json:"approved_by,omitempty"`
	ApprovedAt   string       `json:"approved_at,omitempty"`
}

func (p *BudgetPlan) Clone() *BudgetPlan {
	cp := *p
	if p.Items != nil {
		cp.Items = make([]BudgetItem, len(p.Items))
		copy(cp.Items, p.Items)
	}
	return &cp
}

func (p *BudgetPlan) Recalculate() {
	var planned, actual int64
	for _, item := range p.Items {
		planned += item.Planned
		actual += item.Actual
	}
	p.TotalPlanned = planned
	p.TotalActual = actual
}

func ValidatePlan(p *BudgetPlan) error {
	if p.Name == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if p.FiscalYear < 2020 || p.FiscalYear > 2099 {
		return &core.ValidationError{Field: "fiscal_year", Message: "fiscal year must be 2020-2099"}
	}
	if p.State == "" {
		p.State = BudgetStateDraft
	}
	if !p.State.IsValid() {
		return &core.ValidationError{Field: "state", Message: "invalid state"}
	}
	for _, item := range p.Items {
		if item.Planned < 0 {
			return &core.ValidationError{Field: "items", Message: "planned amount must be non-negative"}
		}
		if item.Actual < 0 {
			return &core.ValidationError{Field: "items", Message: "actual amount must be non-negative"}
		}
		if item.CategoryCode == "" {
			return &core.ValidationError{Field: "items", Message: "category code required"}
		}
	}
	return nil
}

func (s BudgetState) IsValid() bool {
	switch s {
	case BudgetStateDraft, BudgetStateApproved, BudgetStateLocked:
		return true
	default:
		return false
	}
}

type Repository interface {
	Create(ctx context.Context, p *BudgetPlan) error
	FindByID(ctx context.Context, id string) (*BudgetPlan, error)
	Update(ctx context.Context, p *BudgetPlan) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, fiscalYear int, department string) ([]*BudgetPlan, error)
	NextCode(ctx context.Context, fiscalYear int) (int64, error)
}
