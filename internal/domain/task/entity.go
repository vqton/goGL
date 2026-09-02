package task

import (
	"context"
	"time"
)

// JobRun is a single execution record of a task.
type JobRun struct {
	ID          string    `json:"id"`
	TaskName    string    `json:"task_name"`
	Status      string    `json:"status"`  // running | success | failed
	Trigger     string    `json:"trigger"` // manual | scheduled
	TriggeredBy string    `json:"triggered_by"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// TaskDef describes a registered, runnable task.
type TaskDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// HandlerFunc executes a task.
type HandlerFunc func(ctx context.Context) error

type Repository interface {
	CreateRun(ctx context.Context, r *JobRun) error
	FindRun(ctx context.Context, id string) (*JobRun, error)
	UpdateRun(ctx context.Context, r *JobRun) error
	ListRuns(ctx context.Context) ([]*JobRun, error)
}
