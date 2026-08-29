package system

import (
	"context"
	"time"

	"goGL/internal/domain/system"
)

// HealthCheck pings the database. Satisfied by the system persistence repo.
type HealthCheck interface {
	Ping(ctx context.Context) error
}

type Service interface {
	GetInfo(ctx context.Context) (*system.Info, error)
}

type service struct {
	version    string
	commit     string
	goVersion  string
	startedAt  time.Time
	health     HealthCheck
	sessions   system.SessionCounter
	backups    system.LastBackupProvider
}

func NewService(version, commit, goVersion string, startedAt time.Time, health HealthCheck, sessions system.SessionCounter, backups system.LastBackupProvider) Service {
	return &service{
		version:   version,
		commit:    commit,
		goVersion: goVersion,
		startedAt: startedAt,
		health:    health,
		sessions:  sessions,
		backups:   backups,
	}
}

func (s *service) GetInfo(ctx context.Context) (*system.Info, error) {
	info := &system.Info{
		Version:   s.version,
		Commit:    s.commit,
		GoVersion: s.goVersion,
		StartedAt: s.startedAt.UTC().Format(time.RFC3339),
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
	}

	// Best-effort secondary reads: a failure must not take down /system/info.
	info.DBOK = s.health.Ping(ctx) == nil
	if n, err := s.sessions.CountActive(ctx); err == nil {
		info.SessionCount = n
	}
	if ts, err := s.backups.LastBackupAt(ctx); err == nil {
		info.LastBackupAt = ts
	}
	return info, nil
}
