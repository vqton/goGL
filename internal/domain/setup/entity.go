package setup

import (
	"context"

	"goGL/internal/domain/core"
)

type CompanyProfile struct {
	ID               string `json:"id" bson:"_id"`
	Name             string `json:"name" bson:"name"`
	TaxCode          string `json:"tax_code" bson:"tax_code"`
	FiscalYearStart  string `json:"fiscal_year_start" bson:"fiscal_year_start"`
	AccountingRegime string `json:"accounting_regime" bson:"accounting_regime"`
}

type InitialBalance struct {
	ID          string      `json:"id" bson:"_id"`
	AccountCode string      `json:"account_code" bson:"account_code"`
	Period      core.Period `json:"period" bson:"period"`
	Debit       core.Money  `json:"debit" bson:"debit"`
	Credit      core.Money  `json:"credit" bson:"credit"`
}

type Repository interface {
	SaveProfile(ctx context.Context, p *CompanyProfile) error
	GetProfile(ctx context.Context, id string) (*CompanyProfile, error)
	SaveBalance(ctx context.Context, b *InitialBalance) error
}
