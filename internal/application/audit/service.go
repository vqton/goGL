package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"goGL/internal/domain/audit"
)

type Service interface {
	Record(ctx context.Context, l *audit.AuditLog) error
	GetLog(ctx context.Context, id string) (*audit.AuditLog, error)
	ListRecent(ctx context.Context, module string, limit int) ([]*audit.AuditLog, error)
}

type service struct {
	repo audit.Repository
}

func NewService(repo audit.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Record(ctx context.Context, l *audit.AuditLog) error {
	if l.ID == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		l.ID = hex.EncodeToString(b)
	}
	if l.Timestamp == "" {
		l.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return s.repo.Create(ctx, l)
}

func (s *service) GetLog(ctx context.Context, id string) (*audit.AuditLog, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) ListRecent(ctx context.Context, module string, limit int) ([]*audit.AuditLog, error) {
	return s.repo.ListRecent(ctx, module, limit)
}
