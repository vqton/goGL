package tools

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound  = errors.New("tools: not found")
	ErrDuplicate = errors.New("tools: duplicate")
	ErrInvalid   = errors.New("tools: invalid input")
	ErrConflict  = errors.New("tools: conflict")
)

type CardState string

const (
	StateActive   CardState = "active"
	StateInactive CardState = "inactive"
	StateDisposed CardState = "disposed"
)

type ToolCard struct {
	// Core identification
	ID   string `json:"id"`
	Code string `json:"code"` // Auto-generated: TL-XXXXX
	Name string `json:"name"`

	// Classification
	Category    string `json:"category"` // scaffolding, formwork, tools, office_supplies, clothing, etc.
	SubCategory string `json:"sub_category,omitempty"`
	Description string `json:"description,omitempty"`

	// Financial (VND) - must be < 30M VND per Thông tư 99/2025/TT-BTC
	OriginalCost int64  `json:"original_cost"`
	Quantity     int    `json:"quantity"` // Default 1
	Unit         string `json:"unit"`     // pcs, set, pair, etc.

	// Source documents
	PurchaseDate string `json:"purchase_date"`
	InvoiceNo    string `json:"invoice_no,omitempty"`
	Supplier     string `json:"supplier,omitempty"`

	// Location & Assignment
	Warehouse  string `json:"warehouse,omitempty"`
	Department string `json:"department,omitempty"`
	AssignedTo string `json:"assigned_to,omitempty"`
	Location   string `json:"location,omitempty"`

	// Status
	State CardState `json:"state"`

	// GL Integration - Account 153 per Thông tư 99/2025/TT-BTC
	AccountCode153 string `json:"account_code_153,omitempty"` // Account 153 detail
	AccountCodeExp string `json:"account_code_exp,omitempty"` // Expense account (623/627/641/642)

	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedBy string `json:"updated_by,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

func (t *ToolCard) Clone() *ToolCard {
	cp := *t
	return &cp
}

func ValidateToolCard(t *ToolCard) error {
	if t.Name == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if t.OriginalCost <= 0 {
		return &core.ValidationError{Field: "original_cost", Message: "original cost must be positive"}
	}
	// VND 30 million threshold per Thông tư 99/2025/TT-BTC
	// Items >= 30M VND should be tracked as Fixed Assets (TSCĐ)
	if t.OriginalCost >= 30_000_000 {
		return &core.ValidationError{Field: "original_cost", Message: "value must be < 30M VND (use fixedasset module for items >= 30M VND)"}
	}
	if t.Quantity <= 0 {
		t.Quantity = 1
	}
	if t.PurchaseDate == "" {
		return &core.ValidationError{Field: "purchase_date", Message: "purchase date is required"}
	}
	if t.State == "" {
		t.State = StateActive
	}
	if !t.State.IsValid() {
		return &core.ValidationError{Field: "state", Message: "invalid state"}
	}
	return nil
}

func (s CardState) IsValid() bool {
	switch s {
	case StateActive, StateInactive, StateDisposed:
		return true
	default:
		return false
	}
}

type Repository interface {
	Create(ctx context.Context, t *ToolCard) error
	FindByID(ctx context.Context, id string) (*ToolCard, error)
	Update(ctx context.Context, t *ToolCard) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, category string, state CardState, limit, offset int) ([]*ToolCard, error)
	NextCode(ctx context.Context) (int64, error)

	// Transaction operations
	CreateTransaction(ctx context.Context, tx *ToolTransaction) error
	FindTransactionByID(ctx context.Context, id string) (*ToolTransaction, error)
	ListTransactions(ctx context.Context, toolCardID string, txType TransactionType, limit, offset int) ([]*ToolTransaction, error)

	// Inventory operations
	GetStock(ctx context.Context, toolCardID string) (int, error)
	AdjustStock(ctx context.Context, toolCardID string, quantity int) error
	// DecrementStock atomically decrements quantity if sufficient stock exists.
	// Returns ErrInsufficientStock if quantity < requested.
	DecrementStock(ctx context.Context, toolCardID string, quantity int) error
	IncrementStock(ctx context.Context, toolCardID string, quantity int) error
}
