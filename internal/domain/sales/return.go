package sales

import (
	"goGL/internal/domain/core"
)

// ReturnReason represents why goods are being returned.
type ReturnReason string

const (
	ReturnDefective ReturnReason = "defective"
	ReturnWrongItem ReturnReason = "wrong_item"
	ReturnDamaged   ReturnReason = "damaged"
	ReturnCustomer  ReturnReason = "customer_request"
	ReturnExpired   ReturnReason = "expired"
)

// ReturnStatus represents the lifecycle status of a sales return.
type ReturnStatus string

const (
	ReturnDraft     ReturnStatus = "draft"
	ReturnApproved  ReturnStatus = "approved"
	ReturnReceived  ReturnStatus = "received"
	ReturnIssued    ReturnStatus = "credit_note_issued"
	ReturnCompleted ReturnStatus = "completed"
)

func (s ReturnStatus) IsValid() bool {
	switch s {
	case ReturnDraft, ReturnApproved, ReturnReceived, ReturnIssued, ReturnCompleted:
		return true
	default:
		return false
	}
}

// ReturnLine represents a single line item on a sales return.
type ReturnLine struct {
	LineNo      int    `json:"line_no"`
	ItemCode    string `json:"item_code"`
	ItemName    string `json:"item_name"`
	Unit        string `json:"unit"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Amount      int64  `json:"amount"`
	VATAmount   int64  `json:"vat_amount"`
	TotalAmount int64  `json:"total_amount"`
}

// SalesReturn represents a sales return (Trả hàng bán).
type SalesReturn struct {
	ID           string       `json:"id"`
	RefNo        string       `json:"ref_no"` // Auto: PH-XXXXX
	InvoiceID    string       `json:"invoice_id"`
	CustomerCode string       `json:"customer_code"`
	ReturnDate   string       `json:"return_date"`
	Reason       ReturnReason `json:"reason"`
	Status       ReturnStatus `json:"status"`
	Lines        []ReturnLine `json:"lines"`
	SubTotal     core.Money   `json:"sub_total"`
	VATAmount    core.Money   `json:"vat_amount"`
	TotalAmount  core.Money   `json:"total_amount"`
	CreditNoteNo string       `json:"credit_note_no"`
	GLPosted     bool         `json:"gl_posted"`
	GLReference  string       `json:"gl_reference,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	CreatedBy    string       `json:"created_by,omitempty"`
	CreatedAt    string       `json:"created_at"`
	UpdatedBy    string       `json:"updated_by,omitempty"`
	UpdatedAt    string       `json:"updated_at"`
}

func (r *SalesReturn) Clone() *SalesReturn {
	cp := *r
	if r.Lines != nil {
		cp.Lines = make([]ReturnLine, len(r.Lines))
		copy(cp.Lines, r.Lines)
	}
	return &cp
}

// ValidateSalesReturn validates return data.
func ValidateSalesReturn(r *SalesReturn) error {
	if r.InvoiceID == "" {
		return &core.ValidationError{Field: "invoice_id", Message: "invoice ID is required"}
	}
	if r.CustomerCode == "" {
		return &core.ValidationError{Field: "customer_code", Message: "customer code is required"}
	}
	if len(r.Lines) == 0 {
		return ErrEmptyLines
	}
	if r.ReturnDate == "" {
		return &core.ValidationError{Field: "return_date", Message: "return date is required"}
	}
	if r.Reason == "" {
		return &core.ValidationError{Field: "reason", Message: "return reason is required"}
	}
	if r.Status == "" {
		r.Status = ReturnDraft
	}
	if !r.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	for _, line := range r.Lines {
		if line.ItemCode == "" {
			return &core.ValidationError{Field: "item_code", Message: "item code is required"}
		}
		if line.Quantity <= 0 {
			return &core.ValidationError{Field: "quantity", Message: "quantity must be positive"}
		}
	}
	return nil
}
