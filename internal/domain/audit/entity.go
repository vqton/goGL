package audit

import (
	"context"
)

type AuditLog struct {
	ID        string `json:"id" bson:"_id"`
	UserCode  string `json:"user_code" bson:"user_code"`
	Module    string `json:"module" bson:"module"`
	Action    string `json:"action" bson:"action"`
	TargetID  string `json:"target_id" bson:"target_id"`
	Timestamp string `json:"timestamp" bson:"timestamp"`
}

type Repository interface {
	Create(ctx context.Context, l *AuditLog) error
	FindByID(ctx context.Context, id string) (*AuditLog, error)
}
