package tools

import (
	"errors"
)

var (
	ErrInsufficientStock = errors.New("tools: insufficient stock")
	ErrInvalidQuantity   = errors.New("tools: invalid quantity")
)

// TransactionType represents the type of tool transaction.
type TransactionType string

const (
	TxImport     TransactionType = "import"     // Nhập kho
	TxExport     TransactionType = "export"     // Xuất kho
	TxTransfer   TransactionType = "transfer"   // Điều chuyển
	TxReturn     TransactionType = "return"     // Trả lại NCC
	TxDisposal   TransactionType = "disposal"   // Thanh lý
	TxAdjustment TransactionType = "adjustment" // Kiểm kê điều chỉnh
)

func (t TransactionType) IsValid() bool {
	switch t {
	case TxImport, TxExport, TxTransfer, TxReturn, TxDisposal, TxAdjustment:
		return true
	default:
		return false
	}
}

// ToolTransaction represents a transaction for tool card movements.
type ToolTransaction struct {
	ID              string          `json:"id"`
	ToolCardID      string          `json:"tool_card_id"`
	ToolCardCode    string          `json:"tool_card_code"`
	TransactionType TransactionType `json:"transaction_type"`
	Quantity        int             `json:"quantity"`
	UnitPrice       int64           `json:"unit_price"`
	TotalAmount     int64           `json:"total_amount"`
	FromLocation    string          `json:"from_location,omitempty"`
	ToLocation      string          `json:"to_location,omitempty"`
	FromDepartment  string          `json:"from_department,omitempty"`
	ToDepartment    string          `json:"to_department,omitempty"`
	AssignedTo      string          `json:"assigned_to,omitempty"`
	ReferenceNo     string          `json:"reference_no,omitempty"` // Invoice, voucher
	Reason          string          `json:"reason,omitempty"`
	Notes           string          `json:"notes,omitempty"`

	// GL Posting
	GLPosted    bool   `json:"gl_posted"`
	GLReference string `json:"gl_reference,omitempty"`

	// Audit
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}
