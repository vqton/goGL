package tools

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/tools"
)

type Service interface {
	CreateCard(ctx context.Context, t *tools.ToolsCard) error
	GetCard(ctx context.Context, id string) (*tools.ToolsCard, error)
}

type service struct {
	repo tools.Repository
}

func NewService(repo tools.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateCard(ctx context.Context, t *tools.ToolsCard) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetCard(ctx context.Context, id string) (*tools.ToolsCard, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
