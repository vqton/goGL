package reporting

import (
	"context"

	"goGL/internal/domain/core"
)

type FinancialReport struct {
	ID         string      `json:"id" bson:"_id"`
	ReportType string      `json:"report_type" bson:"report_type"`
	Title      string      `json:"title" bson:"title"`
	Period     core.Period `json:"period" bson:"period"`
	Status     core.Status `json:"status" bson:"status"`
}

type DashboardConfig struct {
	ID    string   `json:"id" bson:"_id"`
	Name  string   `json:"name" bson:"name"`
	KPIs  []string `json:"kpis" bson:"kpis"`
	Owner string   `json:"owner" bson:"owner"`
}

type Repository interface {
	Create(ctx context.Context, r *FinancialReport) error
	FindByID(ctx context.Context, id string) (*FinancialReport, error)
	Update(ctx context.Context, r *FinancialReport) error
}
