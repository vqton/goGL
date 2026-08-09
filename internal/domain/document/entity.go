package document

import (
	"context"

	"goGL/internal/domain/core"
)

type Document struct {
	ID     string      `json:"id" bson:"_id"`
	Folder string      `json:"folder" bson:"folder"`
	Name   string      `json:"name" bson:"name"`
	URL    string      `json:"url" bson:"url"`
	Owner  string      `json:"owner" bson:"owner"`
	Status core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, d *Document) error
	FindByID(ctx context.Context, id string) (*Document, error)
	Update(ctx context.Context, d *Document) error
}
