package inventory

import (
	"context"
	"errors"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound          = errors.New("inventory: not found")
	ErrDuplicate         = errors.New("inventory: duplicate")
	ErrInvalid           = errors.New("inventory: invalid input")
	ErrConflict          = errors.New("inventory: conflict")
	ErrInvalidStatus     = errors.New("inventory: invalid status")
	ErrInsufficientStock = errors.New("inventory: insufficient stock")
)

// ItemCategory classifies inventory items per VAS 02.
type ItemCategory string

const (
	CategoryRawMaterials  ItemCategory = "raw_materials"
	CategorySupplies      ItemCategory = "supplies"
	CategoryFinishedGoods ItemCategory = "finished_goods"
	CategoryWIP           ItemCategory = "wip"
	CategoryConsignment   ItemCategory = "consignment"
)

func (c ItemCategory) IsValid() bool {
	switch c {
	case CategoryRawMaterials, CategorySupplies, CategoryFinishedGoods, CategoryWIP, CategoryConsignment:
		return true
	default:
		return false
	}
}

// ValuationMethod defines the inventory cost flow assumption per VAS 02.
// LIFO is explicitly prohibited per Circular 99/2025/TT-BTC.
type ValuationMethod string

const (
	ValuationFIFO            ValuationMethod = "fifo"
	ValuationWeightedAverage ValuationMethod = "weighted_average"
)

func (v ValuationMethod) IsValid() bool {
	switch v {
	case ValuationFIFO, ValuationWeightedAverage:
		return true
	default:
		return false
	}
}

// ItemStatus represents the lifecycle status of an inventory item.
type ItemStatus string

const (
	ItemActive       ItemStatus = "active"
	ItemInactive     ItemStatus = "inactive"
	ItemDiscontinued ItemStatus = "discontinued"
)

func (s ItemStatus) IsValid() bool {
	switch s {
	case ItemActive, ItemInactive, ItemDiscontinued:
		return true
	default:
		return false
	}
}

// Item represents a stock-keeping unit (SKU) per VAS 02.
type Item struct {
	ID              string          `json:"id"`
	Code            string          `json:"code"` // Auto-generated: MH-XXXXX
	Name            string          `json:"name"`
	Category        ItemCategory    `json:"category"`
	SubCategory     string          `json:"sub_category,omitempty"`
	Description     string          `json:"description,omitempty"`
	Unit            string          `json:"unit"` // kg, pcs, m, set, etc.
	ValuationMethod ValuationMethod `json:"valuation_method"`
	GLAccount152    string          `json:"gl_account_152"` // 1521/1522/1523/1524/1526/1527/1528/1529/1531
	GLAccount632    string          `json:"gl_account_632"` // COGS: 6321/6322
	MinStock        float64         `json:"min_stock"`
	MaxStock        float64         `json:"max_stock"`
	ReorderQty      float64         `json:"reorder_qty"`
	Status          ItemStatus      `json:"status"`
	CreatedBy       string          `json:"created_by,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedBy       string          `json:"updated_by,omitempty"`
	UpdatedAt       string          `json:"updated_at"`
}

func (i *Item) Clone() *Item {
	cp := *i
	return &cp
}

// ValidateItem validates an Item per VAS 02 and Thông tư 99/2025/TT-BTC.
func ValidateItem(i *Item) error {
	if i.Name == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if i.Unit == "" {
		return &core.ValidationError{Field: "unit", Message: "unit is required"}
	}
	if i.Category == "" {
		return &core.ValidationError{Field: "category", Message: "category is required"}
	}
	if !i.Category.IsValid() {
		return &core.ValidationError{Field: "category", Message: "invalid category"}
	}
	if i.ValuationMethod == "" {
		i.ValuationMethod = ValuationFIFO
	}
	if !i.ValuationMethod.IsValid() {
		return &core.ValidationError{Field: "valuation_method", Message: "invalid valuation method (LIFO is prohibited)"}
	}
	if i.GLAccount152 == "" {
		return &core.ValidationError{Field: "gl_account_152", Message: "GL account 152 is required"}
	}
	if i.Status == "" {
		i.Status = ItemActive
	}
	if !i.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// WarehouseStatus represents the lifecycle status of a warehouse.
type WarehouseStatus string

const (
	WarehouseActive   WarehouseStatus = "active"
	WarehouseInactive WarehouseStatus = "inactive"
)

func (s WarehouseStatus) IsValid() bool {
	switch s {
	case WarehouseActive, WarehouseInactive:
		return true
	default:
		return false
	}
}

// WarehouseType classifies warehouse purpose.
type WarehouseType string

const (
	WarehouseTypeRawMaterial WarehouseType = "raw_material"
	WarehouseTypeFinished    WarehouseType = "finished_goods"
	WarehouseTypeGeneral     WarehouseType = "general"
)

func (t WarehouseType) IsValid() bool {
	switch t {
	case WarehouseTypeRawMaterial, WarehouseTypeFinished, WarehouseTypeGeneral:
		return true
	default:
		return false
	}
}

// Warehouse represents a storage location.
type Warehouse struct {
	ID            string          `json:"id"`
	Code          string          `json:"code"` // Auto-generated: KHO-XXX
	Name          string          `json:"name"`
	Address       string          `json:"address,omitempty"`
	WarehouseType WarehouseType   `json:"warehouse_type"`
	Manager       string          `json:"manager,omitempty"`
	Phone         string          `json:"phone,omitempty"`
	Status        WarehouseStatus `json:"status"`
	CreatedBy     string          `json:"created_by,omitempty"`
	CreatedAt     string          `json:"created_at"`
	UpdatedBy     string          `json:"updated_by,omitempty"`
	UpdatedAt     string          `json:"updated_at"`
}

func (w *Warehouse) Clone() *Warehouse {
	cp := *w
	return &cp
}

// ValidateWarehouse validates a Warehouse.
func ValidateWarehouse(w *Warehouse) error {
	if w.Name == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if w.WarehouseType == "" {
		w.WarehouseType = WarehouseTypeGeneral
	}
	if !w.WarehouseType.IsValid() {
		return &core.ValidationError{Field: "warehouse_type", Message: "invalid warehouse type"}
	}
	if w.Status == "" {
		w.Status = WarehouseActive
	}
	if !w.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// StockCard represents a running balance per item per warehouse (Sổ kho).
type StockCard struct {
	ID               string  `json:"id"`
	ItemCode         string  `json:"item_code"`
	WarehouseCode    string  `json:"warehouse_code"`
	OpeningQty       float64 `json:"opening_qty"`
	OpeningValue     int64   `json:"opening_value"`
	TotalInQty       float64 `json:"total_in_qty"`
	TotalInValue     int64   `json:"total_in_value"`
	TotalOutQty      float64 `json:"total_out_qty"`
	TotalOutValue    int64   `json:"total_out_value"`
	CurrentQty       float64 `json:"current_qty"`
	CurrentValue     int64   `json:"current_value"`
	AverageCost      int64   `json:"average_cost"`
	LastMovementID   string  `json:"last_movement_id,omitempty"`
	LastMovementDate string  `json:"last_movement_date,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

// StockValuationLayer tracks an individual receipt batch for FIFO valuation.
// Each receipt creates a layer; dispatches consume layers oldest-first.
type StockValuationLayer struct {
	ID            string  `json:"id"`
	ItemCode      string  `json:"item_code"`
	WarehouseCode string  `json:"warehouse_code"`
	MovementID    string  `json:"movement_id"`
	Quantity      float64 `json:"quantity"`
	UnitCost      int64   `json:"unit_cost"`
	TotalCost     int64   `json:"total_cost"`
	RemainingQty  float64 `json:"remaining_qty"`
	ReceivedDate  string  `json:"received_date"`
	ExpiryDate    string  `json:"expiry_date,omitempty"`
}

func (l *StockValuationLayer) Clone() *StockValuationLayer {
	cp := *l
	return &cp
}

// DispatchQty returns the quantity to consume from this layer (min of remaining and requested).
func (l *StockValuationLayer) DispatchQty(requested float64) float64 {
	if l.RemainingQty <= 0 {
		return 0
	}
	if requested >= l.RemainingQty {
		return l.RemainingQty
	}
	return requested
}

// MovementType classifies the direction and purpose of a stock movement.
type MovementType string

const (
	MovementReceipt         MovementType = "receipt"
	MovementDispatch        MovementType = "dispatch"
	MovementTransferIn      MovementType = "transfer_in"
	MovementTransferOut     MovementType = "transfer_out"
	MovementAdjustmentPlus  MovementType = "adjustment_plus"
	MovementAdjustmentMinus MovementType = "adjustment_minus"
	MovementOpeningBalance  MovementType = "opening_balance"
)

func (t MovementType) IsValid() bool {
	switch t {
	case MovementReceipt, MovementDispatch, MovementTransferIn, MovementTransferOut,
		MovementAdjustmentPlus, MovementAdjustmentMinus, MovementOpeningBalance:
		return true
	default:
		return false
	}
}

// IsInbound returns true for movement types that increase stock.
func (t MovementType) IsInbound() bool {
	switch t {
	case MovementReceipt, MovementTransferIn, MovementAdjustmentPlus, MovementOpeningBalance:
		return true
	default:
		return false
	}
}

// IsOutbound returns true for movement types that decrease stock.
func (t MovementType) IsOutbound() bool {
	switch t {
	case MovementDispatch, MovementTransferOut, MovementAdjustmentMinus:
		return true
	default:
		return false
	}
}

// MovementStatus represents the lifecycle status of a stock movement.
type MovementStatus string

const (
	MovementDraft     MovementStatus = "draft"
	MovementConfirmed MovementStatus = "confirmed"
	MovementPosted    MovementStatus = "posted"
	MovementCancelled MovementStatus = "cancelled"
)

func (s MovementStatus) IsValid() bool {
	switch s {
	case MovementDraft, MovementConfirmed, MovementPosted, MovementCancelled:
		return true
	default:
		return false
	}
}

// StockMovement represents a single stock transaction (receipt, dispatch, transfer, adjustment).
type StockMovement struct {
	ID            string         `json:"id"`
	MovementCode  string         `json:"movement_code"` // Auto: PN-XXXXX, PX-XXXXX, CC-XXXXX, DK-XXXXX
	MovementType  MovementType   `json:"movement_type"`
	MovementDate  string         `json:"movement_date"`
	ItemCode      string         `json:"item_code"`
	WarehouseCode string         `json:"warehouse_code"`
	Quantity      float64        `json:"quantity"`
	UnitPrice     int64          `json:"unit_price"`
	TotalCost     int64          `json:"total_cost"`
	RefDocType    string         `json:"ref_doc_type,omitempty"`
	RefDocID      string         `json:"ref_doc_id,omitempty"`
	RefDocNo      string         `json:"ref_doc_no,omitempty"`
	FromWarehouse string         `json:"from_warehouse,omitempty"`
	ToWarehouse   string         `json:"to_warehouse,omitempty"`
	GLPosted      bool           `json:"gl_posted"`
	GLJournalID   string         `json:"gl_journal_id,omitempty"`
	Status        MovementStatus `json:"status"`
	CreatedBy     string         `json:"created_by,omitempty"`
	CreatedAt     string         `json:"created_at"`
	ConfirmedBy   string         `json:"confirmed_by,omitempty"`
	ConfirmedAt   string         `json:"confirmed_at,omitempty"`
}

func (m *StockMovement) Clone() *StockMovement {
	cp := *m
	return &cp
}

// ValidateStockMovement validates a StockMovement.
func ValidateStockMovement(m *StockMovement) error {
	if m.MovementType == "" {
		return &core.ValidationError{Field: "movement_type", Message: "movement type is required"}
	}
	if !m.MovementType.IsValid() {
		return &core.ValidationError{Field: "movement_type", Message: "invalid movement type"}
	}
	if m.ItemCode == "" {
		return &core.ValidationError{Field: "item_code", Message: "item code is required"}
	}
	if m.WarehouseCode == "" {
		return &core.ValidationError{Field: "warehouse_code", Message: "warehouse code is required"}
	}
	if m.Quantity <= 0 {
		return &core.ValidationError{Field: "quantity", Message: "quantity must be positive"}
	}
	if m.UnitPrice < 0 {
		return &core.ValidationError{Field: "unit_price", Message: "unit price must be non-negative"}
	}
	if m.MovementDate == "" {
		return &core.ValidationError{Field: "movement_date", Message: "movement date is required"}
	}
	if m.Status == "" {
		m.Status = MovementDraft
	}
	if !m.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// PhysicalCountStatus represents the lifecycle of a physical inventory count.
type PhysicalCountStatus string

const (
	PhysicalCountDraft      PhysicalCountStatus = "draft"
	PhysicalCountInProgress PhysicalCountStatus = "in_progress"
	PhysicalCountCompleted  PhysicalCountStatus = "completed"
	PhysicalCountReconciled PhysicalCountStatus = "reconciled"
)

func (s PhysicalCountStatus) IsValid() bool {
	switch s {
	case PhysicalCountDraft, PhysicalCountInProgress, PhysicalCountCompleted, PhysicalCountReconciled:
		return true
	default:
		return false
	}
}

// PhysicalCount represents a physical inventory count session per warehouse.
type PhysicalCount struct {
	ID            string              `json:"id"`
	CountCode     string              `json:"count_code"` // Auto: PC-XXXXX
	WarehouseCode string              `json:"warehouse_code"`
	CountDate     string              `json:"count_date"`
	Status        PhysicalCountStatus `json:"status"`
	Lines         []PhysicalCountLine `json:"lines,omitempty"`
	Notes         string              `json:"notes,omitempty"`
	CreatedBy     string              `json:"created_by,omitempty"`
	CreatedAt     string              `json:"created_at"`
	CompletedBy   string              `json:"completed_by,omitempty"`
	CompletedAt   string              `json:"completed_at,omitempty"`
}

func (p *PhysicalCount) Clone() *PhysicalCount {
	cp := *p
	if p.Lines != nil {
		cp.Lines = make([]PhysicalCountLine, len(p.Lines))
		copy(cp.Lines, p.Lines)
	}
	return &cp
}

// PhysicalCountLine represents one item counted in a physical count.
type PhysicalCountLine struct {
	ID             string  `json:"id"`
	ItemCode       string  `json:"item_code"`
	SystemQty      float64 `json:"system_qty"`
	CountedQty     float64 `json:"counted_qty"`
	Difference     float64 `json:"difference"`
	UnitCost       int64   `json:"unit_cost"`
	AdjustmentType string  `json:"adjustment_type,omitempty"` // "plus" or "minus" or ""
}

// ValidatePhysicalCount validates a PhysicalCount.
func ValidatePhysicalCount(p *PhysicalCount) error {
	if p.WarehouseCode == "" {
		return &core.ValidationError{Field: "warehouse_code", Message: "warehouse code is required"}
	}
	if p.CountDate == "" {
		return &core.ValidationError{Field: "count_date", Message: "count date is required"}
	}
	if p.Status == "" {
		p.Status = PhysicalCountDraft
	}
	if !p.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// Repository defines the persistence interface for inventory entities.
type Repository interface {
	// Item CRUD
	CreateItem(ctx context.Context, i *Item) error
	FindItemByID(ctx context.Context, id string) (*Item, error)
	FindItemByCode(ctx context.Context, code string) (*Item, error)
	UpdateItem(ctx context.Context, i *Item) error
	DeleteItem(ctx context.Context, id string) error
	ListItems(ctx context.Context, category ItemCategory, status ItemStatus, search string, limit, offset int) ([]*Item, int, error)
	NextItemCode(ctx context.Context) (int64, error)

	// Warehouse CRUD
	CreateWarehouse(ctx context.Context, w *Warehouse) error
	FindWarehouseByID(ctx context.Context, id string) (*Warehouse, error)
	FindWarehouseByCode(ctx context.Context, code string) (*Warehouse, error)
	UpdateWarehouse(ctx context.Context, w *Warehouse) error
	DeleteWarehouse(ctx context.Context, id string) error
	ListWarehouses(ctx context.Context, status WarehouseStatus, limit, offset int) ([]*Warehouse, int, error)
	NextWarehouseCode(ctx context.Context) (int64, error)

	// Stock Card operations
	GetStockCard(ctx context.Context, itemCode, warehouseCode string) (*StockCard, error)
	ListStockCards(ctx context.Context, warehouseCode string, limit, offset int) ([]*StockCard, int, error)
	UpsertStockCard(ctx context.Context, sc *StockCard) error

	// Stock Movement operations
	CreateMovement(ctx context.Context, m *StockMovement) error
	FindMovementByID(ctx context.Context, id string) (*StockMovement, error)
	UpdateMovement(ctx context.Context, m *StockMovement) error
	ListMovements(ctx context.Context, itemCode, warehouseCode string, movementType MovementType, limit, offset int) ([]*StockMovement, int, error)
	NextMovementCode(ctx context.Context, prefix string) (int64, error)

	// Valuation Layer operations (FIFO)
	CreateLayer(ctx context.Context, l *StockValuationLayer) error
	UpdateLayer(ctx context.Context, l *StockValuationLayer) error
	ListLayersByItem(ctx context.Context, itemCode, warehouseCode string) ([]*StockValuationLayer, error)

	// Physical Count operations
	CreatePhysicalCount(ctx context.Context, pc *PhysicalCount) error
	FindPhysicalCountByID(ctx context.Context, id string) (*PhysicalCount, error)
	UpdatePhysicalCount(ctx context.Context, pc *PhysicalCount) error
	ListPhysicalCounts(ctx context.Context, warehouseCode string, status PhysicalCountStatus, limit, offset int) ([]*PhysicalCount, int, error)
	NextPhysicalCountCode(ctx context.Context) (int64, error)
}
