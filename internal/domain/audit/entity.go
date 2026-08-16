package audit

import (
	"context"
)

type AuditLog struct {
	ID        string `json:"id"`
	UserCode  string `json:"user_code"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	TargetID  string `json:"target_id"`
	Timestamp string `json:"timestamp"`
}

type Repository interface {
	Create(ctx context.Context, l *AuditLog) error
	FindByID(ctx context.Context, id string) (*AuditLog, error)
	ListRecent(ctx context.Context, module string, limit int) ([]*AuditLog, error)
}
