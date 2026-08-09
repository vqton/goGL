package budget

import (
	"context"

	"goGL/internal/domain/core"
)

type BudgetItem struct {
	CategoryCode string     `json:"category_code" bson:"category_code"`
	Planned      core.Money `json:"planned" bson:"planned"`
	Actual       core.Money `json:"actual" bson:"actual"`
}

type BudgetPlan struct {
	ID       string       `json:"id" bson:"_id"`
	PlanCode string       `json:"plan_code" bson:"plan_code"`
	Period   core.Period  `json:"period" bson:"period"`
	Items    []BudgetItem `json:"items" bson:"items"`
	Status   core.Status  `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, p *BudgetPlan) error
	FindByID(ctx context.Context, id string) (*BudgetPlan, error)
	Update(ctx context.Context, p *BudgetPlan) error
}
