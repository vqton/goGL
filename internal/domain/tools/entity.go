package tools

import (
	"context"

	"goGL/internal/domain/core"
)

type ToolsCard struct {
	ID       string      `json:"id" bson:"_id"`
	Code     string      `json:"code" bson:"code"`
	Name     string      `json:"name" bson:"name"`
	Quantity float64     `json:"quantity" bson:"quantity"`
	Cost     core.Money  `json:"cost" bson:"cost"`
	Status   core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, t *ToolsCard) error
	FindByID(ctx context.Context, id string) (*ToolsCard, error)
	Update(ctx context.Context, t *ToolsCard) error
}
