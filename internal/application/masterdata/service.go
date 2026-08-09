package masterdata

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/masterdata"
)

type Service interface {
	CreateItem(ctx context.Context, i *masterdata.CatalogItem) error
	GetItem(ctx context.Context, id string) (*masterdata.CatalogItem, error)
}

type service struct {
	repo masterdata.Repository
}

func NewService(repo masterdata.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateItem(ctx context.Context, i *masterdata.CatalogItem) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetItem(ctx context.Context, id string) (*masterdata.CatalogItem, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
