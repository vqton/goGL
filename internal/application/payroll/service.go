package payroll

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/payroll"
)

type Service interface {
	CreatePayslip(ctx context.Context, p *payroll.Payslip) error
	GetPayslip(ctx context.Context, id string) (*payroll.Payslip, error)
}

type service struct {
	repo payroll.Repository
}

func NewService(repo payroll.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreatePayslip(ctx context.Context, p *payroll.Payslip) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetPayslip(ctx context.Context, id string) (*payroll.Payslip, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}
