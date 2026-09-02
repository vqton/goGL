package tax

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound  = errors.New("tax: not found")
	ErrDuplicate = errors.New("tax: duplicate")
	ErrInvalid   = errors.New("tax: invalid input")
	ErrLocked    = errors.New("tax: declaration is locked")
)

type TaxType string

const (
	TaxTypeVAT            TaxType = "vat"
	TaxTypeCIT            TaxType = "cit"
	TaxTypePersonalIncome TaxType = "pit"
	TaxTypeImportDuty     TaxType = "import_duty"
	TaxTypeExcise         TaxType = "excise"
	TaxTypeOther          TaxType = "other"
)

type DeclarationState string

const (
	StateDraft    DeclarationState = "draft"
	StateFiled    DeclarationState = "filed"
	StateApproved DeclarationState = "approved"
	StateRejected DeclarationState = "rejected"
)

type LineItem struct {
	Description string `json:"description"`
	TaxRate     int64  `json:"tax_rate"` // basis points (100 = 1%)
	Taxable     int64  `json:"taxable"`
	TaxAmount   int64  `json:"tax_amount"`
	Notes       string `json:"notes,omitempty"`
}

type TaxDeclaration struct {
	ID           string           `json:"id"`
	Code         string           `json:"code"`
	TaxType      TaxType          `json:"tax_type"`
	Period       string           `json:"period"` // e.g. "2026-Q1" or "2026-01"
	CompanyID    string           `json:"company_id,omitempty"`
	TotalTaxable int64            `json:"total_taxable"`
	TotalTax     int64            `json:"total_tax"`
	RefNo        string           `json:"ref_no,omitempty"`
	State        DeclarationState `json:"state"`
	Items        []LineItem       `json:"items"`
	Notes        string           `json:"notes,omitempty"`
	CreatedBy    string           `json:"created_by,omitempty"`
	CreatedAt    string           `json:"created_at"`
	UpdatedBy    string           `json:"updated_by,omitempty"`
	UpdatedAt    string           `json:"updated_at"`
	FiledBy      string           `json:"filed_by,omitempty"`
	FiledAt      string           `json:"filed_at,omitempty"`
}

func (d *TaxDeclaration) Clone() *TaxDeclaration {
	cp := *d
	if d.Items != nil {
		cp.Items = make([]LineItem, len(d.Items))
		copy(cp.Items, d.Items)
	}
	return &cp
}

func (d *TaxDeclaration) Recalculate() {
	var taxable, tax int64
	for _, item := range d.Items {
		taxable += item.Taxable
		tax += item.TaxAmount
	}
	d.TotalTaxable = taxable
	d.TotalTax = tax
}

func ValidateDeclaration(d *TaxDeclaration) error {
	if !d.TaxType.IsValid() {
		return &core.ValidationError{Field: "tax_type", Message: "invalid tax type"}
	}
	if d.Period == "" {
		return &core.ValidationError{Field: "period", Message: "period is required"}
	}
	if d.State == "" {
		d.State = StateDraft
	}
	if !d.State.IsValid() {
		return &core.ValidationError{Field: "state", Message: "invalid state"}
	}
	for _, item := range d.Items {
		if item.TaxRate < 0 || item.TaxRate > 10000 {
			return &core.ValidationError{Field: "items", Message: "tax_rate must be 0-10000 basis points"}
		}
		if item.Taxable < 0 {
			return &core.ValidationError{Field: "items", Message: "taxable amount must be non-negative"}
		}
	}
	return nil
}

func (t TaxType) IsValid() bool {
	switch t {
	case TaxTypeVAT, TaxTypeCIT, TaxTypePersonalIncome, TaxTypeImportDuty, TaxTypeExcise, TaxTypeOther:
		return true
	default:
		return false
	}
}

func (s DeclarationState) IsValid() bool {
	switch s {
	case StateDraft, StateFiled, StateApproved, StateRejected:
		return true
	default:
		return false
	}
}

func TaxTypeCode(t TaxType) string {
	switch t {
	case TaxTypeVAT:
		return "VAT"
	case TaxTypeCIT:
		return "CIT"
	case TaxTypePersonalIncome:
		return "PIT"
	case TaxTypeImportDuty:
		return "IMP"
	case TaxTypeExcise:
		return "EXC"
	default:
		return "OTH"
	}
}

type Repository interface {
	Create(ctx context.Context, d *TaxDeclaration) error
	FindByID(ctx context.Context, id string) (*TaxDeclaration, error)
	Update(ctx context.Context, d *TaxDeclaration) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, taxType TaxType, period string) ([]*TaxDeclaration, error)
	NextCode(ctx context.Context, taxType TaxType) (int64, error)
}
