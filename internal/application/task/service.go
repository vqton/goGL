package task

import (
	"context"
	"errors"
	"sync"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/task"
)

// Auditor records task executions. Satisfied by the audit service.
type Auditor interface {
	Record(ctx context.Context, l *audit.AuditLog) error
}

type Service interface {
	ListTasks(ctx context.Context) ([]*task.TaskDef, error)
	RunNow(ctx context.Context, by, name string) (*task.JobRun, error)
	ListRuns(ctx context.Context) ([]*task.JobRun, error)
}

// Registry maps task names to their handlers.
type Registry map[string]task.HandlerFunc

type service struct {
	registry Registry
	repo     task.Repository
	auditor  Auditor
	now      func() time.Time
	mu       sync.Mutex
	running  map[string]bool
}

func NewService(registry Registry, repo task.Repository, auditor Auditor) Service {
	return &service{
		registry: registry,
		repo:     repo,
		auditor:  auditor,
		now:      time.Now,
		running:  make(map[string]bool),
	}
}

func (s *service) ListTasks(ctx context.Context) ([]*task.TaskDef, error) {
	out := make([]*task.TaskDef, 0, len(s.registry))
	for name := range s.registry {
		out = append(out, &task.TaskDef{Name: name, Enabled: true})
	}
	return out, nil
}

func (s *service) ListRuns(ctx context.Context) ([]*task.JobRun, error) {
	return s.repo.ListRuns(ctx)
}

func (s *service) RunNow(ctx context.Context, by, name string) (*task.JobRun, error) {
	handler, ok := s.registry[name]
	if !ok {
		return nil, task.ErrUnknown
	}

	s.mu.Lock()
	if s.running[name] {
		s.mu.Unlock()
		return nil, task.ErrInProgress
	}
	s.running[name] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, name)
		s.mu.Unlock()
	}()

	now := s.now().UTC()
	run := &task.JobRun{
		ID:          "tr_" + now.Format("20060102150405") + "_" + name,
		TaskName:    name,
		Status:      "running",
		Trigger:     "manual",
		TriggeredBy: by,
		StartedAt:   now,
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	runErr := handler(ctx)
	finished := s.now().UTC()
	run.FinishedAt = finished
	if runErr != nil {
		run.Status = "failed"
		run.Error = runErr.Error()
	} else {
		run.Status = "success"
	}
	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return nil, err
	}

	_ = s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "task", Action: "task." + run.Status,
		TargetID: run.ID, Timestamp: finished.Format(time.RFC3339),
	})
	if runErr != nil {
		return nil, errors.New(run.Error)
	}
	return run, nil
}
