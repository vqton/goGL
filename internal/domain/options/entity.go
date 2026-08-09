package options

import (
	"context"
)

type Option struct {
	ID       string `json:"id" bson:"_id"`
	Key      string `json:"key" bson:"key"`
	Value    string `json:"value" bson:"value"`
	Category string `json:"category" bson:"category"`
}

type Repository interface {
	Create(ctx context.Context, o *Option) error
	FindByKey(ctx context.Context, key string) (*Option, error)
	Update(ctx context.Context, o *Option) error
}
