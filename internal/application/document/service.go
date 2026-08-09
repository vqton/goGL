package document

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/document"
)

type Service interface {
	CreateDocument(ctx context.Context, d *document.Document) error
	GetDocument(ctx context.Context, id string) (*document.Document, error)
}

type service struct {
	repo document.Repository
}

func NewService(repo document.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateDocument(ctx context.Context, d *document.Document) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetDocument(ctx context.Context, id string) (*document.Document, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
