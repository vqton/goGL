package options

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/options"
)

type Service interface {
	SetOption(ctx context.Context, o *options.Option) error
	GetOption(ctx context.Context, key string) (*options.Option, error)
}

type service struct {
	repo options.Repository
}

func NewService(repo options.Repository) Service {
	return &service{repo: repo}
}

func (s *service) SetOption(ctx context.Context, o *options.Option) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetOption(ctx context.Context, key string) (*options.Option, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
