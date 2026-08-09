package task

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/task"
)

type Service interface {
	CreateTask(ctx context.Context, t *task.Task) error
	GetTask(ctx context.Context, id string) (*task.Task, error)
}

type service struct {
	repo task.Repository
}

func NewService(repo task.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateTask(ctx context.Context, t *task.Task) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetTask(ctx context.Context, id string) (*task.Task, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
