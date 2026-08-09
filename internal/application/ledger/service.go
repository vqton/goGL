package ledger

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/ledger"
)

type Service interface {
	CreateEntry(ctx context.Context, e *ledger.JournalEntry) error
	GetEntry(ctx context.Context, id string) (*ledger.JournalEntry, error)
}

type service struct {
	repo ledger.Repository
}

func NewService(repo ledger.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateEntry(ctx context.Context, e *ledger.JournalEntry) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetEntry(ctx context.Context, id string) (*ledger.JournalEntry, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
