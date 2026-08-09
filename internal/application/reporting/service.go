package reporting

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/reporting"
)

type Service interface {
	CreateReport(ctx context.Context, r *reporting.FinancialReport) error
	GetReport(ctx context.Context, id string) (*reporting.FinancialReport, error)
}

type service struct {
	repo reporting.Repository
}

func NewService(repo reporting.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateReport(ctx context.Context, r *reporting.FinancialReport) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetReport(ctx context.Context, id string) (*reporting.FinancialReport, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
