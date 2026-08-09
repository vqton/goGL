package setup

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/setup"
)

type Service interface {
	Initialize(ctx context.Context, p *setup.CompanyProfile) error
	GetProfile(ctx context.Context, id string) (*setup.CompanyProfile, error)
	ImportOpeningBalances(ctx context.Context, b *setup.InitialBalance) error
}

type service struct {
	repo setup.Repository
}

func NewService(repo setup.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Initialize(ctx context.Context, p *setup.CompanyProfile) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetProfile(ctx context.Context, id string) (*setup.CompanyProfile, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}

func (s *service) ImportOpeningBalances(ctx context.Context, b *setup.InitialBalance) error {
	// TODO: implement
	return core.ErrNotImplemented
}
