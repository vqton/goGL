package fixedasset

import (
	"context"
	"errors"
	"time"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound           = errors.New("fixedasset: not found")
	ErrDuplicate          = errors.New("fixedasset: duplicate")
	ErrInvalid            = errors.New("fixedasset: invalid input")
	ErrConflict           = errors.New("fixedasset: conflict")
	ErrLiquidationPending = errors.New("fixedasset: liquidation pending")
	ErrFullyDepreciated   = errors.New("fixedasset: fully depreciated")
)

// AssetState represents the lifecycle state of a fixed asset.
type AssetState string

const (
	StateActive             AssetState = "active"
	StateInactive           AssetState = "inactive"
	StateScrapped           AssetState = "scrapped"
	StateSold               AssetState = "sold"
	StatePendingLiquidation AssetState = "pending_liquidation"
)

// DepreciationMethod - only 3 methods per Circular 45/2013/TT-BTC.
type DepreciationMethod string

const (
	MethodStraightLine  DepreciationMethod = "straight_line"
	MethodDeclining     DepreciationMethod = "declining_balance"
	MethodUnitsOfOutput DepreciationMethod = "units_of_output"
)

// AssetType - 6 types per Circular 45/2013/TT-BTC Article 6.
type AssetType string

const (
	TypeHousing   AssetType = "housing"   // Nhà cửa, vật kiến trúc
	TypeMachinery AssetType = "machinery" // Máy móc, thiết bị
	TypeTransport AssetType = "transport" // Phương tiện vận tải
	TypeTools     AssetType = "tools"     // Dụng cụ quản lý
	TypePerennial AssetType = "perennial" // Vườn cây lâu năm, súc vật
	TypeOther     AssetType = "other"     // Các loại khác
)

// FixedAsset represents a tangible fixed asset per VAS 03 and Circular 45/2013.
type FixedAsset struct {
	// Core identification
	ID   string `json:"id"`
	Code string `json:"code"` // Auto-generated: FA-XXXXX
	Name string `json:"name"`

	// Classification per Circular 45 Article 6
	AssetType   AssetType `json:"asset_type"`
	Category    string    `json:"category,omitempty"` // Sub-category within type
	Description string    `json:"description,omitempty"`

	// Financial (VND)
	OriginalCost    int64 `json:"original_cost"`
	ResidualValue   int64 `json:"residual_value"`
	AccumulatedDepr int64 `json:"accumulated_depr"`

	// Depreciation settings
	DepreciationMethod DepreciationMethod `json:"depreciation_method"`
	UsefulLifeMonths   int                `json:"useful_life_months"`

	// Dates
	PurchaseDate   string `json:"purchase_date"`
	InServiceDate  string `json:"in_service_date"`
	LastReviewDate string `json:"last_review_date,omitempty"`

	// Location and assignment
	Location   string `json:"location,omitempty"`
	Department string `json:"department,omitempty"`

	// Source documents
	VendorName      string `json:"vendor_name,omitempty"`
	PurchaseOrderNo string `json:"purchase_order_no,omitempty"`
	InvoiceNo       string `json:"invoice_no,omitempty"`
	SerialNo        string `json:"serial_no,omitempty"`

	// Accounting
	AccountCode211 string `json:"account_code_211,omitempty"` // Original cost account
	AccountCode214 string `json:"account_code_214,omitempty"` // Accumulated depreciation account

	// State
	State AssetState `json:"state"`

	// Audit
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedBy string `json:"updated_by,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

func (a *FixedAsset) Clone() *FixedAsset {
	cp := *a
	return &cp
}

// CurrentValue returns OriginalCost - AccumulatedDepr.
func (a *FixedAsset) CurrentValue() int64 {
	return a.OriginalCost - a.AccumulatedDepr
}

// IsFullyDepreciated returns true if accumulated depreciation equals or exceeds depreciable value.
func (a *FixedAsset) IsFullyDepreciated() bool {
	return a.AccumulatedDepr >= (a.OriginalCost - a.ResidualValue)
}

// ValidateAsset validates asset data per Circular 45 and VAS 03.
func ValidateAsset(a *FixedAsset) error {
	if a.Name == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if a.OriginalCost < 30_000_000 { // VND 30 million minimum per Circular 45
		return &core.ValidationError{Field: "original_cost", Message: "original cost must be >= 30,000,000 VND"}
	}
	if a.ResidualValue < 0 {
		return &core.ValidationError{Field: "residual_value", Message: "residual value must be non-negative"}
	}
	if a.ResidualValue >= a.OriginalCost {
		return &core.ValidationError{Field: "residual_value", Message: "residual value must be less than original cost"}
	}
	if a.UsefulLifeMonths <= 0 {
		return &core.ValidationError{Field: "useful_life_months", Message: "useful life must be positive"}
	}
	if !a.AssetType.IsValid() {
		return &core.ValidationError{Field: "asset_type", Message: "invalid asset type"}
	}
	if !a.DepreciationMethod.IsValid() {
		return &core.ValidationError{Field: "depreciation_method", Message: "invalid depreciation method"}
	}
	if a.PurchaseDate == "" {
		return &core.ValidationError{Field: "purchase_date", Message: "purchase date is required"}
	}
	if _, err := time.Parse("2006-01-02", a.PurchaseDate); err != nil {
		return &core.ValidationError{Field: "purchase_date", Message: "purchase date must be YYYY-MM-DD"}
	}
	if a.InServiceDate == "" {
		return &core.ValidationError{Field: "in_service_date", Message: "in-service date is required"}
	}
	if _, err := time.Parse("2006-01-02", a.InServiceDate); err != nil {
		return &core.ValidationError{Field: "in_service_date", Message: "in-service date must be YYYY-MM-DD"}
	}
	if a.State == "" {
		a.State = StateActive
	}
	if !a.State.IsValid() {
		return &core.ValidationError{Field: "state", Message: "invalid state"}
	}
	return nil
}

func (s AssetState) IsValid() bool {
	switch s {
	case StateActive, StateInactive, StateScrapped, StateSold, StatePendingLiquidation:
		return true
	default:
		return false
	}
}

func (m DepreciationMethod) IsValid() bool {
	switch m {
	case MethodStraightLine, MethodDeclining, MethodUnitsOfOutput:
		return true
	default:
		return false
	}
}

func (t AssetType) IsValid() bool {
	switch t {
	case TypeHousing, TypeMachinery, TypeTransport, TypeTools, TypePerennial, TypeOther:
		return true
	default:
		return false
	}
}

// DepreciationLifeRange returns min/max months for asset type per Annex I of Circular 45/2013.
func DepreciationLifeRange(t AssetType) (minMonths, maxMonths int) {
	switch t {
	case TypeHousing:
		return 60, 600 // 5-50 years
	case TypeMachinery:
		return 36, 240 // 3-20 years
	case TypeTransport:
		return 72, 360 // 6-30 years
	case TypeTools:
		return 36, 120 // 3-10 years
	case TypePerennial:
		return 48, 480 // 4-40 years
	default:
		return 48, 300 // 4-25 years
	}
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Repository defines the persistence interface for fixed assets.
type Repository interface {
	Create(ctx context.Context, a *FixedAsset) error
	FindByID(ctx context.Context, id string) (*FixedAsset, error)
	Update(ctx context.Context, a *FixedAsset) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, assetType AssetType, state AssetState, limit, offset int) ([]*FixedAsset, error)
	NextCode(ctx context.Context) (int64, error)
}
