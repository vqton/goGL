package costing

import (
	"context"

	"goGL/internal/domain/core"
)

type CostSheet struct {
	ID          string      `json:"id" bson:"_id"`
	ProductCode string      `json:"product_code" bson:"product_code"`
	Period      core.Period `json:"period" bson:"period"`
	TotalCost   core.Money  `json:"total_cost" bson:"total_cost"`
	UnitCost    core.Money  `json:"unit_cost" bson:"unit_cost"`
	Status      core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, c *CostSheet) error
	FindByID(ctx context.Context, id string) (*CostSheet, error)
	Update(ctx context.Context, c *CostSheet) error
}
