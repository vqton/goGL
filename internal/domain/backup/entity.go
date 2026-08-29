package backup

import (
	"context"
	"time"
)

// BackupArtifact is one VACUUM INTO snapshot stored on disk with its
// SHA-256 checksum for integrity verification.
type BackupArtifact struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
	SHA256    string    `json:"sha256"`
	Tier      string    `json:"tier"`
	Trigger   string    `json:"trigger"` // manual | scheduled
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// BackupJob is the scheduled backup configuration (cron expression).
type BackupJob struct {
	ID        string    `json:"id"`
	Schedule  string    `json:"schedule"`
	TargetDir string    `json:"target_dir"`
	Enabled   bool      `json:"enabled"`
	LastRunAt time.Time `json:"last_run_at"`
	LastError string    `json:"last_error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RestorePlan is a staged restore awaiting approval. Two-step restore:
// stage copies the artifact and verifies it (integrity_check + SHA-256),
// approve swaps it over the live database.
type RestorePlan struct {
	ID         string    `json:"id"`
	ArtifactID string    `json:"artifact_id"`
	StagedPath string    `json:"staged_path"`
	Status     string    `json:"status"` // staged | approved
	Integrity  string    `json:"integrity"` // ok | <error>
	RowCount   int       `json:"row_count"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"`
}

type Repository interface {
	CreateArtifact(ctx context.Context, a *BackupArtifact) error
	FindArtifact(ctx context.Context, id string) (*BackupArtifact, error)
	ListArtifacts(ctx context.Context) ([]*BackupArtifact, error)
	DeleteArtifact(ctx context.Context, id string) error
	CreateJob(ctx context.Context, j *BackupJob) error
	FindJob(ctx context.Context, id string) (*BackupJob, error)
	UpdateJob(ctx context.Context, j *BackupJob) error
	CreatePlan(ctx context.Context, p *RestorePlan) error
	FindPlan(ctx context.Context, id string) (*RestorePlan, error)
	UpdatePlan(ctx context.Context, p *RestorePlan) error
}
