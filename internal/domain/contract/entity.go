package contract

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound  = errors.New("contract: not found")
	ErrDuplicate = errors.New("contract: duplicate")
	ErrInvalid   = errors.New("contract: invalid input")
	ErrLocked    = errors.New("contract: contract is locked")
)

type ContractType string

const (
	TypeService      ContractType = "service"
	TypePurchase     ContractType = "purchase"
	TypeSales        ContractType = "sales"
	TypeEmployment   ContractType = "employment"
	TypeLease        ContractType = "lease"
	TypeLoan         ContractType = "loan"
	TypeOther        ContractType = "other"
)

type ContractState string

const (
	StateDraft     ContractState = "draft"
	StateActive    ContractState = "active"
	StateExpired   ContractState = "expired"
	StateTerminated ContractState = "terminated"
)

type Contract struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Type        ContractType   `json:"type"`
	PartyName   string         `json:"party_name"`
	PartyTaxID  string         `json:"party_tax_id,omitempty"`
	Value       int64          `json:"value"`
	Currency    string         `json:"currency"`
	StartDate   string         `json:"start_date"`
	EndDate     string         `json:"end_date"`
	State       ContractState  `json:"state"`
	Notes       string         `json:"notes,omitempty"`
	CreatedBy   string         `json:"created_by,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedBy   string         `json:"updated_by,omitempty"`
	UpdatedAt   string         `json:"updated_at"`
}

func (c *Contract) Clone() *Contract {
	cp := *c
	return &cp
}

func ValidateContract(c *Contract) error {
	if c.Name == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if c.PartyName == "" {
		return &core.ValidationError{Field: "party_name", Message: "party name is required"}
	}
	if !c.Type.IsValid() {
		return &core.ValidationError{Field: "type", Message: "invalid contract type"}
	}
	if c.StartDate == "" {
		return &core.ValidationError{Field: "start_date", Message: "start date is required"}
	}
	if c.EndDate == "" {
		return &core.ValidationError{Field: "end_date", Message: "end date is required"}
	}
	if c.StartDate > c.EndDate {
		return &core.ValidationError{Field: "end_date", Message: "end date must be after start date"}
	}
	if c.Value < 0 {
		return &core.ValidationError{Field: "value", Message: "value must be non-negative"}
	}
	if c.State == "" {
		c.State = StateDraft
	}
	if !c.State.IsValid() {
		return &core.ValidationError{Field: "state", Message: "invalid state"}
	}
	if c.Currency == "" {
		c.Currency = "VND"
	}
	return nil
}

func (t ContractType) IsValid() bool {
	switch t {
	case TypeService, TypePurchase, TypeSales, TypeEmployment, TypeLease, TypeLoan, TypeOther:
		return true
	default:
		return false
	}
}

func (s ContractState) IsValid() bool {
	switch s {
	case StateDraft, StateActive, StateExpired, StateTerminated:
		return true
	default:
		return false
	}
}

type LoanAgreement struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	ContractID       string `json:"contract_id,omitempty"`
	Principal        int64  `json:"principal"`
	InterestRateBP   int64  `json:"interest_rate_bp"`
	TermMonths       int    `json:"term_months"`
	MonthlyPayment   int64  `json:"monthly_payment"`
	DisbursementDate string `json:"disbursement_date,omitempty"`
	Status           string `json:"status"`
	CreatedBy        string `json:"created_by,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedBy        string `json:"updated_by,omitempty"`
	UpdatedAt        string `json:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, c *Contract) error
	FindByID(ctx context.Context, id string) (*Contract, error)
	Update(ctx context.Context, c *Contract) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, contractType ContractType, state ContractState) ([]*Contract, error)
	NextCode(ctx context.Context) (int64, error)
	CreateLoan(ctx context.Context, l *LoanAgreement) error
	FindLoanByID(ctx context.Context, id string) (*LoanAgreement, error)
	UpdateLoan(ctx context.Context, l *LoanAgreement) error
}
