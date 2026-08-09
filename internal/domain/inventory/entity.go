package inventory

import (
	"context"

	"goGL/internal/domain/core"
)

type StockMovement struct {
	ID            string      `json:"id" bson:"_id"`
	ItemCode      string      `json:"item_code" bson:"item_code"`
	WarehouseCode string      `json:"warehouse_code" bson:"warehouse_code"`
	RefDate       string      `json:"ref_date" bson:"ref_date"`
	Quantity      float64     `json:"quantity" bson:"quantity"`
	UnitPrice     core.Money  `json:"unit_price" bson:"unit_price"`
	Status        core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, m *StockMovement) error
	FindByID(ctx context.Context, id string) (*StockMovement, error)
	Update(ctx context.Context, m *StockMovement) error
}
