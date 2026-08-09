package contract

import (
	"context"

	"goGL/internal/domain/core"
)

type Contract struct {
	ID        string      `json:"id" bson:"_id"`
	Code      string      `json:"code" bson:"code"`
	PartyName string      `json:"party_name" bson:"party_name"`
	StartDate string      `json:"start_date" bson:"start_date"`
	EndDate   string      `json:"end_date" bson:"end_date"`
	Amount    core.Money  `json:"amount" bson:"amount"`
	Status    core.Status `json:"status" bson:"status"`
}

type LoanAgreement struct {
	ID           string      `json:"id" bson:"_id"`
	Code         string      `json:"code" bson:"code"`
	LenderName   string      `json:"lender_name" bson:"lender_name"`
	Principal    core.Money  `json:"principal" bson:"principal"`
	InterestRate float64     `json:"interest_rate" bson:"interest_rate"`
	Status       core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, c *Contract) error
	FindByID(ctx context.Context, id string) (*Contract, error)
	Update(ctx context.Context, c *Contract) error
	CreateLoan(ctx context.Context, l *LoanAgreement) error
}
