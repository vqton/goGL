package payroll

import (
	"context"

	"goGL/internal/domain/core"
)

type Payslip struct {
	ID           string      `json:"id" bson:"_id"`
	EmployeeCode string      `json:"employee_code" bson:"employee_code"`
	Period       core.Period `json:"period" bson:"period"`
	GrossSalary  core.Money  `json:"gross_salary" bson:"gross_salary"`
	NetSalary    core.Money  `json:"net_salary" bson:"net_salary"`
	Status       core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, p *Payslip) error
	FindByID(ctx context.Context, id string) (*Payslip, error)
	Update(ctx context.Context, p *Payslip) error
}
