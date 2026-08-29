package bank

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound  = errors.New("bank: not found")
	ErrDuplicate = errors.New("bank: duplicate")
	ErrInvalid   = errors.New("bank: invalid input")
	ErrConflict  = errors.New("bank: conflict")
)

type TransactionType string

const (
	TxTypeDeposit   TransactionType = "deposit"
	TxTypeWithdrawal TransactionType = "withdrawal"
	TxTypeTransfer  TransactionType = "transfer"
	TxTypeFee       TransactionType = "fee"
	TxTypeInterest  TransactionType = "interest"
	TxTypeOther     TransactionType = "other"
)

type TransactionState string

const (
	StatePending   TransactionState = "pending"
	StateCleared   TransactionState = "cleared"
	StateReconciled TransactionState = "reconciled"
	StateCancelled TransactionState = "cancelled"
)

type BankTransaction struct {
	ID            string          `json:"id"`
	Code          string          `json:"code"`
	AccountNo     string          `json:"account_no"`
	BankCode      string          `json:"bank_code"`
	BankName      string          `json:"bank_name,omitempty"`
	RefDate       string          `json:"ref_date"`
	ValueDate     string          `json:"value_date,omitempty"`
	Amount        int64           `json:"amount"`
	Currency      string          `json:"currency,omitempty"`
	Type          TransactionType `json:"type"`
	State         TransactionState `json:"state"`
	Description   string          `json:"description,omitempty"`
	Counterparty  string          `json:"counterparty,omitempty"`
	Reference     string          `json:"reference,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	CreatedBy     string          `json:"created_by,omitempty"`
	CreatedAt     string          `json:"created_at"`
	UpdatedBy     string          `json:"updated_by,omitempty"`
	UpdatedAt     string          `json:"updated_at"`
}

func (t *BankTransaction) Clone() *BankTransaction {
	cp := *t
	return &cp
}

func ValidateTransaction(t *BankTransaction) error {
	if t.AccountNo == "" {
		return &core.ValidationError{Field: "account_no", Message: "account number is required"}
	}
	if t.BankCode == "" {
		return &core.ValidationError{Field: "bank_code", Message: "bank code is required"}
	}
	if t.RefDate == "" {
		return &core.ValidationError{Field: "ref_date", Message: "reference date is required"}
	}
	if t.Amount == 0 {
		return &core.ValidationError{Field: "amount", Message: "amount must not be zero"}
	}
	if !t.Type.IsValid() {
		return &core.ValidationError{Field: "type", Message: "invalid transaction type"}
	}
	if t.State == "" {
		t.State = StatePending
	}
	if !t.State.IsValid() {
		return &core.ValidationError{Field: "state", Message: "invalid state"}
	}
	if t.Currency == "" {
		t.Currency = "VND"
	}
	return nil
}

func (t TransactionType) IsValid() bool {
	switch t {
	case TxTypeDeposit, TxTypeWithdrawal, TxTypeTransfer, TxTypeFee, TxTypeInterest, TxTypeOther:
		return true
	default:
		return false
	}
}

func (s TransactionState) IsValid() bool {
	switch s {
	case StatePending, StateCleared, StateReconciled, StateCancelled:
		return true
	default:
		return false
	}
}

type Repository interface {
	Create(ctx context.Context, t *BankTransaction) error
	FindByID(ctx context.Context, id string) (*BankTransaction, error)
	Update(ctx context.Context, t *BankTransaction) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, accountNo string, txType TransactionType) ([]*BankTransaction, error)
	NextCode(ctx context.Context) (int64, error)
}
