package fixedasset

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/fixedasset"
)

type Service interface {
	CreateAsset(ctx context.Context, a *fixedasset.FixedAsset) error
	GetAsset(ctx context.Context, id string) (*fixedasset.FixedAsset, error)
}

type service struct {
	repo fixedasset.Repository
}

func NewService(repo fixedasset.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateAsset(ctx context.Context, a *fixedasset.FixedAsset) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetAsset(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
