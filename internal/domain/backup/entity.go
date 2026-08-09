package backup

import (
	"context"

	"goGL/internal/domain/core"
)

type BackupJob struct {
	ID       string      `json:"id" bson:"_id"`
	Schedule string      `json:"schedule" bson:"schedule"`
	Target   string      `json:"target" bson:"target"`
	LastRun  string      `json:"last_run" bson:"last_run"`
	Status   core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, j *BackupJob) error
	FindByID(ctx context.Context, id string) (*BackupJob, error)
	Update(ctx context.Context, j *BackupJob) error
}
