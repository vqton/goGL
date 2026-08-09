package system

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/system"
)

type Service interface {
	CreateTenant(ctx context.Context, t *system.Tenant) error
	GetTenant(ctx context.Context, id string) (*system.Tenant, error)
}

type service struct {
	repo system.Repository
}

func NewService(repo system.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTenant(ctx context.Context, t *system.Tenant) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetTenant(ctx context.Context, id string) (*system.Tenant, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
