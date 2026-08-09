package system

import (
	"context"

	"goGL/internal/domain/core"
)

type Tenant struct {
	ID     string      `json:"id" bson:"_id"`
	Code   string      `json:"code" bson:"code"`
	Name   string      `json:"name" bson:"name"`
	Status core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, t *Tenant) error
	FindByID(ctx context.Context, id string) (*Tenant, error)
	Update(ctx context.Context, t *Tenant) error
}
