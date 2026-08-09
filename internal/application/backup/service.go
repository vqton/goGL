package backup

import (
	"context"

	"goGL/internal/domain/backup"
	"goGL/internal/domain/core"
)

type Service interface {
	CreateJob(ctx context.Context, j *backup.BackupJob) error
	GetJob(ctx context.Context, id string) (*backup.BackupJob, error)
	Restore(ctx context.Context, id string) error
}

type service struct {
	repo backup.Repository
}

func NewService(repo backup.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateJob(ctx context.Context, j *backup.BackupJob) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetJob(ctx context.Context, id string) (*backup.BackupJob, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}

func (s *service) Restore(ctx context.Context, id string) error {
	// TODO: implement
	return core.ErrNotImplemented
}
