package system

import "context"

// Info is the runtime health/identity snapshot served by GET /system/info.
type Info struct {
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	GoVersion     string `json:"go_version"`
	StartedAt     string `json:"started_at"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	DBOK          bool   `json:"db_ok"`
	SessionCount  int    `json:"session_count"`
	LastBackupAt  string `json:"last_backup_at"`
}

// SessionCounter counts active sessions (for system/info).
type SessionCounter interface {
	CountActive(ctx context.Context) (int, error)
}

// LastBackupProvider returns the timestamp of the newest backup artifact.
type LastBackupProvider interface {
	LastBackupAt(ctx context.Context) (string, error)
}
