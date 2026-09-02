package inventory

import (
	"context"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/inventory"
)

type Service interface {
	// Item operations
	CreateItem(ctx context.Context, i *inventory.Item, actor string) (*inventory.Item, error)
	GetItem(ctx context.Context, id string) (*inventory.Item, error)
	GetItemByCode(ctx context.Context, code string) (*inventory.Item, error)
	UpdateItem(ctx context.Context, id string, patch *inventory.Item, actor string) (*inventory.Item, error)
	DeactivateItem(ctx context.Context, id string) error
	ListItems(ctx context.Context, category inventory.ItemCategory, status inventory.ItemStatus, search string, limit, offset int) ([]*inventory.Item, int, error)

	// Warehouse operations
	CreateWarehouse(ctx context.Context, w *inventory.Warehouse, actor string) (*inventory.Warehouse, error)
	GetWarehouse(ctx context.Context, id string) (*inventory.Warehouse, error)
	GetWarehouseByCode(ctx context.Context, code string) (*inventory.Warehouse, error)
	UpdateWarehouse(ctx context.Context, id string, patch *inventory.Warehouse, actor string) (*inventory.Warehouse, error)
	DeactivateWarehouse(ctx context.Context, id string) error
	ListWarehouses(ctx context.Context, status inventory.WarehouseStatus, limit, offset int) ([]*inventory.Warehouse, int, error)

	// Stock operations
	GetStockBalance(ctx context.Context, itemCode, warehouseCode string) (*inventory.StockCard, error)
	ListStockBalances(ctx context.Context, warehouseCode string, limit, offset int) ([]*inventory.StockCard, int, error)

	// Stock Movement operations
	CreateMovement(ctx context.Context, m *inventory.StockMovement, actor string) (*inventory.StockMovement, error)
	GetMovement(ctx context.Context, id string) (*inventory.StockMovement, error)
	ConfirmMovement(ctx context.Context, id string, actor string) (*inventory.StockMovement, error)
	ListMovements(ctx context.Context, itemCode, warehouseCode string, movementType inventory.MovementType, limit, offset int) ([]*inventory.StockMovement, int, error)

	// Stock Transfer: moves stock between warehouses (2 movements, 1 transaction)
	TransferStock(ctx context.Context, itemCode, fromWarehouse, toWarehouse string, quantity float64, unitPrice int64, movementDate, actor string) (*inventory.StockMovement, *inventory.StockMovement, error)

	// Stock Adjustment: manual correction
	AdjustStock(ctx context.Context, itemCode, warehouseCode string, quantity float64, unitPrice int64, reason, movementDate, actor string) (*inventory.StockMovement, error)

	// Physical Count operations
	CreatePhysicalCount(ctx context.Context, pc *inventory.PhysicalCount, actor string) (*inventory.PhysicalCount, error)
	GetPhysicalCount(ctx context.Context, id string) (*inventory.PhysicalCount, error)
	UpdatePhysicalCount(ctx context.Context, id string, patch *inventory.PhysicalCount, actor string) (*inventory.PhysicalCount, error)
	ListPhysicalCounts(ctx context.Context, warehouseCode string, status inventory.PhysicalCountStatus, limit, offset int) ([]*inventory.PhysicalCount, int, error)
	CompletePhysicalCount(ctx context.Context, id string, actor string) (*inventory.PhysicalCount, error)
	ReconcilePhysicalCount(ctx context.Context, id string, actor string) ([]*inventory.StockMovement, error)

	// NRV Write-Down operations
	WriteDownNRV(ctx context.Context, itemCode, warehouseCode string, nrvUnitCost int64, movementDate, actor string) (*inventory.StockMovement, error)
	ReverseWriteDown(ctx context.Context, itemCode, warehouseCode string, nrvUnitCost int64, movementDate, actor string) (*inventory.StockMovement, error)
}

type service struct {
	repo inventory.Repository
}

func NewService(repo inventory.Repository) Service {
	return &service{repo: repo}
}

// --- Item operations ---

func (s *service) CreateItem(ctx context.Context, i *inventory.Item, actor string) (*inventory.Item, error) {
	i2 := i.Clone()
	i2.CreatedBy = actor
	i2.UpdatedBy = actor

	if err := inventory.ValidateItem(i2); err != nil {
		return nil, err
	}

	n, err := s.repo.NextItemCode(ctx)
	if err != nil {
		return nil, err
	}
	i2.Code = fmt.Sprintf("MH-%05d", n)
	i2.ID = core.RowID("inventory_item", i2.Code)

	now := core.NowRFC3339()
	i2.CreatedAt = now
	i2.UpdatedAt = now

	if err := s.repo.CreateItem(ctx, i2); err != nil {
		return nil, err
	}
	return i2, nil
}

func (s *service) GetItem(ctx context.Context, id string) (*inventory.Item, error) {
	return s.repo.FindItemByID(ctx, id)
}

func (s *service) GetItemByCode(ctx context.Context, code string) (*inventory.Item, error) {
	return s.repo.FindItemByCode(ctx, code)
}

func (s *service) UpdateItem(ctx context.Context, id string, patch *inventory.Item, actor string) (*inventory.Item, error) {
	cur, err := s.repo.FindItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Category != "" {
		cur.Category = patch.Category
	}
	if patch.SubCategory != "" {
		cur.SubCategory = patch.SubCategory
	}
	if patch.Description != "" {
		cur.Description = patch.Description
	}
	if patch.Unit != "" {
		cur.Unit = patch.Unit
	}
	if patch.ValuationMethod != "" {
		cur.ValuationMethod = patch.ValuationMethod
	}
	if patch.GLAccount152 != "" {
		cur.GLAccount152 = patch.GLAccount152
	}
	if patch.GLAccount632 != "" {
		cur.GLAccount632 = patch.GLAccount632
	}
	if patch.MinStock > 0 {
		cur.MinStock = patch.MinStock
	}
	if patch.MaxStock > 0 {
		cur.MaxStock = patch.MaxStock
	}
	if patch.ReorderQty > 0 {
		cur.ReorderQty = patch.ReorderQty
	}
	if patch.Status != "" {
		cur.Status = patch.Status
	}
	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := inventory.ValidateItem(cur); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) DeactivateItem(ctx context.Context, id string) error {
	cur, err := s.repo.FindItemByID(ctx, id)
	if err != nil {
		return err
	}
	cur.Status = inventory.ItemInactive
	cur.UpdatedAt = core.NowRFC3339()
	return s.repo.UpdateItem(ctx, cur)
}

func (s *service) ListItems(ctx context.Context, category inventory.ItemCategory, status inventory.ItemStatus, search string, limit, offset int) ([]*inventory.Item, int, error) {
	return s.repo.ListItems(ctx, category, status, search, limit, offset)
}

// --- Warehouse operations ---

func (s *service) CreateWarehouse(ctx context.Context, w *inventory.Warehouse, actor string) (*inventory.Warehouse, error) {
	w2 := w.Clone()
	w2.CreatedBy = actor
	w2.UpdatedBy = actor

	if err := inventory.ValidateWarehouse(w2); err != nil {
		return nil, err
	}

	n, err := s.repo.NextWarehouseCode(ctx)
	if err != nil {
		return nil, err
	}
	w2.Code = fmt.Sprintf("KHO-%03d", n)
	w2.ID = core.RowID("inventory_warehouse", w2.Code)

	now := core.NowRFC3339()
	w2.CreatedAt = now
	w2.UpdatedAt = now

	if err := s.repo.CreateWarehouse(ctx, w2); err != nil {
		return nil, err
	}
	return w2, nil
}

func (s *service) GetWarehouse(ctx context.Context, id string) (*inventory.Warehouse, error) {
	return s.repo.FindWarehouseByID(ctx, id)
}

func (s *service) GetWarehouseByCode(ctx context.Context, code string) (*inventory.Warehouse, error) {
	return s.repo.FindWarehouseByCode(ctx, code)
}

func (s *service) UpdateWarehouse(ctx context.Context, id string, patch *inventory.Warehouse, actor string) (*inventory.Warehouse, error) {
	cur, err := s.repo.FindWarehouseByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Address != "" {
		cur.Address = patch.Address
	}
	if patch.WarehouseType != "" {
		cur.WarehouseType = patch.WarehouseType
	}
	if patch.Manager != "" {
		cur.Manager = patch.Manager
	}
	if patch.Phone != "" {
		cur.Phone = patch.Phone
	}
	if patch.Status != "" {
		cur.Status = patch.Status
	}
	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := inventory.ValidateWarehouse(cur); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateWarehouse(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) DeactivateWarehouse(ctx context.Context, id string) error {
	cur, err := s.repo.FindWarehouseByID(ctx, id)
	if err != nil {
		return err
	}
	cur.Status = inventory.WarehouseInactive
	cur.UpdatedAt = core.NowRFC3339()
	return s.repo.UpdateWarehouse(ctx, cur)
}

func (s *service) ListWarehouses(ctx context.Context, status inventory.WarehouseStatus, limit, offset int) ([]*inventory.Warehouse, int, error) {
	return s.repo.ListWarehouses(ctx, status, limit, offset)
}

// --- Stock operations ---

func (s *service) GetStockBalance(ctx context.Context, itemCode, warehouseCode string) (*inventory.StockCard, error) {
	return s.repo.GetStockCard(ctx, itemCode, warehouseCode)
}

func (s *service) ListStockBalances(ctx context.Context, warehouseCode string, limit, offset int) ([]*inventory.StockCard, int, error) {
	return s.repo.ListStockCards(ctx, warehouseCode, limit, offset)
}

// --- Stock Movement operations ---

func (s *service) CreateMovement(ctx context.Context, m *inventory.StockMovement, actor string) (*inventory.StockMovement, error) {
	m2 := m.Clone()
	m2.CreatedBy = actor

	if err := inventory.ValidateStockMovement(m2); err != nil {
		return nil, err
	}

	prefix := movementCodePrefix(m2.MovementType)
	n, err := s.repo.NextMovementCode(ctx, prefix)
	if err != nil {
		return nil, err
	}
	m2.MovementCode = fmt.Sprintf("%s-%05d", prefix, n)
	m2.ID = core.RowID("inventory_movement", m2.MovementCode)
	m2.TotalCost = int64(m2.Quantity) * m2.UnitPrice

	now := core.NowRFC3339()
	m2.CreatedAt = now

	if err := s.repo.CreateMovement(ctx, m2); err != nil {
		return nil, err
	}
	return m2, nil
}

func (s *service) GetMovement(ctx context.Context, id string) (*inventory.StockMovement, error) {
	return s.repo.FindMovementByID(ctx, id)
}

func (s *service) ConfirmMovement(ctx context.Context, id string, actor string) (*inventory.StockMovement, error) {
	m, err := s.repo.FindMovementByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.Status != inventory.MovementDraft {
		return nil, inventory.ErrInvalid
	}

	item, err := s.repo.FindItemByCode(ctx, m.ItemCode)
	if err != nil {
		return nil, err
	}

	sc, err := s.repo.GetStockCard(ctx, m.ItemCode, m.WarehouseCode)
	if err != nil && err != inventory.ErrNotFound {
		return nil, err
	}

	now := core.NowRFC3339()
	if sc == nil {
		sc = &inventory.StockCard{
			ID:            stockCardID(m.ItemCode, m.WarehouseCode),
			ItemCode:      m.ItemCode,
			WarehouseCode: m.WarehouseCode,
			AverageCost:   m.UnitPrice,
		}
	}

	if m.MovementType.IsInbound() {
		// Inbound: increase stock card
		sc.CurrentQty += m.Quantity
		sc.TotalInQty += m.Quantity
		sc.TotalInValue += m.TotalCost
		sc.CurrentValue += m.TotalCost

		// FIFO: create a valuation layer
		if item.ValuationMethod == inventory.ValuationFIFO {
			layer := &inventory.StockValuationLayer{
				ID:            core.RowID("val_layer", m.MovementCode, m.ItemCode, m.WarehouseCode),
				ItemCode:      m.ItemCode,
				WarehouseCode: m.WarehouseCode,
				MovementID:    m.ID,
				Quantity:      m.Quantity,
				UnitCost:      m.UnitPrice,
				TotalCost:     m.TotalCost,
				RemainingQty:  m.Quantity,
				ReceivedDate:  m.MovementDate,
			}
			if err := s.repo.CreateLayer(ctx, layer); err != nil {
				return nil, err
			}
		}
	} else {
		// Outbound: decrease stock card
		if sc.CurrentQty < m.Quantity {
			return nil, inventory.ErrInsufficientStock
		}

		if item.ValuationMethod == inventory.ValuationFIFO {
			// FIFO: consume layers oldest-first
			layers, err := s.repo.ListLayersByItem(ctx, m.ItemCode, m.WarehouseCode)
			if err != nil {
				return nil, err
			}
			results, totalCost, err := inventory.FIFOAlloc(layers, m.Quantity)
			if err != nil {
				return nil, err
			}
			// Update remaining qty on each consumed layer
			for i, r := range results {
				for _, l := range layers {
					if l.ID == r.LayerID {
						l.RemainingQty -= results[i].Qty
						if err := s.repo.UpdateLayer(ctx, l); err != nil {
							return nil, err
						}
						break
					}
				}
			}
			m.TotalCost = totalCost
		} else {
			// Weighted Average: reduce proportionally
			if sc.CurrentQty > 0 {
				removedValue := int64(m.Quantity * float64(sc.CurrentValue) / sc.CurrentQty)
				sc.CurrentValue -= removedValue
				m.TotalCost = removedValue
			}
		}

		sc.CurrentQty -= m.Quantity
		sc.TotalOutQty += m.Quantity
		sc.TotalOutValue += m.TotalCost
	}

	// Update average cost
	if sc.CurrentQty > 0 {
		if item.ValuationMethod == inventory.ValuationWeightedAverage && m.MovementType.IsInbound() {
			sc.AverageCost = inventory.WeightedAvgCost(
				sc.CurrentQty-m.Quantity, sc.AverageCost, m.Quantity, m.UnitPrice)
		} else if sc.CurrentQty > 0 {
			sc.AverageCost = sc.CurrentValue / int64(sc.CurrentQty)
		}
	} else {
		sc.AverageCost = m.UnitPrice
	}

	sc.LastMovementID = m.ID
	sc.LastMovementDate = now
	sc.UpdatedAt = now

	if err := s.repo.UpsertStockCard(ctx, sc); err != nil {
		return nil, err
	}

	m.Status = inventory.MovementConfirmed
	m.ConfirmedBy = actor
	m.ConfirmedAt = now
	if err := s.repo.UpdateMovement(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *service) ListMovements(ctx context.Context, itemCode, warehouseCode string, movementType inventory.MovementType, limit, offset int) ([]*inventory.StockMovement, int, error) {
	return s.repo.ListMovements(ctx, itemCode, warehouseCode, movementType, limit, offset)
}

func (s *service) TransferStock(ctx context.Context, itemCode, fromWarehouse, toWarehouse string, quantity float64, unitPrice int64, movementDate, actor string) (*inventory.StockMovement, *inventory.StockMovement, error) {
	if fromWarehouse == toWarehouse {
		return nil, nil, inventory.ErrInvalid
	}

	item, err := s.repo.FindItemByCode(ctx, itemCode)
	if err != nil {
		return nil, nil, err
	}
	_ = item

	// Leg 1: Transfer Out from source
	out := &inventory.StockMovement{
		MovementType:  inventory.MovementTransferOut,
		MovementDate:  movementDate,
		ItemCode:      itemCode,
		WarehouseCode: fromWarehouse,
		Quantity:      quantity,
		UnitPrice:     unitPrice,
		ToWarehouse:   toWarehouse,
	}
	outMov, err := s.CreateMovement(ctx, out, actor)
	if err != nil {
		return nil, nil, err
	}
	if _, err := s.ConfirmMovement(ctx, outMov.ID, actor); err != nil {
		return nil, nil, err
	}
	outMov, err = s.repo.FindMovementByID(ctx, outMov.ID)
	if err != nil {
		return nil, nil, err
	}

	// Leg 2: Transfer In to destination
	in := &inventory.StockMovement{
		MovementType:  inventory.MovementTransferIn,
		MovementDate:  movementDate,
		ItemCode:      itemCode,
		WarehouseCode: toWarehouse,
		Quantity:      quantity,
		UnitPrice:     unitPrice,
		FromWarehouse: fromWarehouse,
	}
	inMov, err := s.CreateMovement(ctx, in, actor)
	if err != nil {
		return nil, nil, err
	}
	if _, err := s.ConfirmMovement(ctx, inMov.ID, actor); err != nil {
		return nil, nil, err
	}
	inMov, err = s.repo.FindMovementByID(ctx, inMov.ID)
	if err != nil {
		return nil, nil, err
	}

	return outMov, inMov, nil
}

func (s *service) AdjustStock(ctx context.Context, itemCode, warehouseCode string, quantity float64, unitPrice int64, reason, movementDate, actor string) (*inventory.StockMovement, error) {
	if reason == "" {
		return nil, inventory.ErrInvalid
	}

	var movType inventory.MovementType
	if quantity > 0 {
		movType = inventory.MovementAdjustmentPlus
	} else {
		movType = inventory.MovementAdjustmentMinus
		quantity = -quantity
	}

	m := &inventory.StockMovement{
		MovementType:  movType,
		MovementDate:  movementDate,
		ItemCode:      itemCode,
		WarehouseCode: warehouseCode,
		Quantity:      quantity,
		UnitPrice:     unitPrice,
	}
	mov, err := s.CreateMovement(ctx, m, actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.ConfirmMovement(ctx, mov.ID, actor); err != nil {
		return nil, err
	}
	return s.repo.FindMovementByID(ctx, mov.ID)
}

// --- Physical Count operations ---

func (s *service) CreatePhysicalCount(ctx context.Context, pc *inventory.PhysicalCount, actor string) (*inventory.PhysicalCount, error) {
	pc2 := pc.Clone()
	pc2.CreatedBy = actor

	if err := inventory.ValidatePhysicalCount(pc2); err != nil {
		return nil, err
	}

	n, err := s.repo.NextPhysicalCountCode(ctx)
	if err != nil {
		return nil, err
	}
	pc2.CountCode = fmt.Sprintf("PC-%05d", n)
	pc2.ID = core.RowID("inventory_physical_count", pc2.CountCode)

	now := core.NowRFC3339()
	pc2.CreatedAt = now

	if err := s.repo.CreatePhysicalCount(ctx, pc2); err != nil {
		return nil, err
	}
	return pc2, nil
}

func (s *service) GetPhysicalCount(ctx context.Context, id string) (*inventory.PhysicalCount, error) {
	return s.repo.FindPhysicalCountByID(ctx, id)
}

func (s *service) UpdatePhysicalCount(ctx context.Context, id string, patch *inventory.PhysicalCount, actor string) (*inventory.PhysicalCount, error) {
	cur, err := s.repo.FindPhysicalCountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status != inventory.PhysicalCountDraft {
		return nil, inventory.ErrInvalid
	}
	if patch.Lines != nil {
		cur.Lines = patch.Lines
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	if err := s.repo.UpdatePhysicalCount(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) ListPhysicalCounts(ctx context.Context, warehouseCode string, status inventory.PhysicalCountStatus, limit, offset int) ([]*inventory.PhysicalCount, int, error) {
	return s.repo.ListPhysicalCounts(ctx, warehouseCode, status, limit, offset)
}

func (s *service) CompletePhysicalCount(ctx context.Context, id string, actor string) (*inventory.PhysicalCount, error) {
	pc, err := s.repo.FindPhysicalCountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pc.Status != inventory.PhysicalCountDraft && pc.Status != inventory.PhysicalCountInProgress {
		return nil, inventory.ErrInvalid
	}

	// Auto-populate system quantities from stock cards
	for i := range pc.Lines {
		sc, err := s.repo.GetStockCard(ctx, pc.Lines[i].ItemCode, pc.WarehouseCode)
		if err != nil && err != inventory.ErrNotFound {
			return nil, err
		}
		if sc != nil {
			pc.Lines[i].SystemQty = sc.CurrentQty
		}
		pc.Lines[i].Difference = pc.Lines[i].CountedQty - pc.Lines[i].SystemQty
		if pc.Lines[i].Difference > 0 {
			pc.Lines[i].AdjustmentType = "plus"
		} else if pc.Lines[i].Difference < 0 {
			pc.Lines[i].AdjustmentType = "minus"
		}
	}

	now := core.NowRFC3339()
	pc.Status = inventory.PhysicalCountCompleted
	pc.CompletedBy = actor
	pc.CompletedAt = now

	if err := s.repo.UpdatePhysicalCount(ctx, pc); err != nil {
		return nil, err
	}
	return pc, nil
}

func (s *service) ReconcilePhysicalCount(ctx context.Context, id string, actor string) ([]*inventory.StockMovement, error) {
	pc, err := s.repo.FindPhysicalCountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pc.Status != inventory.PhysicalCountCompleted {
		return nil, inventory.ErrInvalid
	}

	var movements []*inventory.StockMovement
	for _, line := range pc.Lines {
		if line.Difference == 0 {
			continue
		}

		mov, err := s.AdjustStock(ctx, line.ItemCode, pc.WarehouseCode, line.Difference, line.UnitCost, fmt.Sprintf("Physical count %s", pc.CountCode), pc.CountDate, actor)
		if err != nil {
			return nil, err
		}
		movements = append(movements, mov)
	}

	now := core.NowRFC3339()
	pc.Status = inventory.PhysicalCountReconciled
	pc.CompletedAt = now
	if err := s.repo.UpdatePhysicalCount(ctx, pc); err != nil {
		return nil, err
	}

	return movements, nil
}

// --- NRV Write-Down operations ---

func (s *service) WriteDownNRV(ctx context.Context, itemCode, warehouseCode string, nrvUnitCost int64, movementDate, actor string) (*inventory.StockMovement, error) {
	sc, err := s.repo.GetStockCard(ctx, itemCode, warehouseCode)
	if err != nil {
		return nil, err
	}
	if sc.CurrentQty <= 0 {
		return nil, inventory.ErrInvalid
	}
	if sc.AverageCost <= nrvUnitCost {
		return nil, inventory.ErrInvalid
	}

	writeDownQty := sc.CurrentQty
	unitCostDiff := sc.AverageCost - nrvUnitCost
	totalWriteDown := int64(writeDownQty) * unitCostDiff

	mov := &inventory.StockMovement{
		MovementType:  inventory.MovementAdjustmentMinus,
		MovementDate:  movementDate,
		ItemCode:      itemCode,
		WarehouseCode: warehouseCode,
		Quantity:      writeDownQty,
		UnitPrice:     unitCostDiff,
		TotalCost:     totalWriteDown,
		RefDocType:    "nrv_write_down",
	}
	mov2, err := s.CreateMovement(ctx, mov, actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.ConfirmMovement(ctx, mov2.ID, actor); err != nil {
		return nil, err
	}
	return s.repo.FindMovementByID(ctx, mov2.ID)
}

func (s *service) ReverseWriteDown(ctx context.Context, itemCode, warehouseCode string, nrvUnitCost int64, movementDate, actor string) (*inventory.StockMovement, error) {
	sc, err := s.repo.GetStockCard(ctx, itemCode, warehouseCode)
	if err != nil {
		return nil, err
	}
	if sc.CurrentQty <= 0 {
		return nil, inventory.ErrInvalid
	}

	// Calculate how much to reverse: the difference between current avg cost and the NRV
	// Only reverse if NRV has recovered above current average cost
	if nrvUnitCost <= sc.AverageCost {
		return nil, inventory.ErrInvalid
	}

	reverseQty := sc.CurrentQty
	unitCostDiff := nrvUnitCost - sc.AverageCost
	totalReverse := int64(reverseQty) * unitCostDiff

	mov := &inventory.StockMovement{
		MovementType:  inventory.MovementAdjustmentPlus,
		MovementDate:  movementDate,
		ItemCode:      itemCode,
		WarehouseCode: warehouseCode,
		Quantity:      reverseQty,
		UnitPrice:     unitCostDiff,
		TotalCost:     totalReverse,
		RefDocType:    "nrv_reversal",
	}
	mov2, err := s.CreateMovement(ctx, mov, actor)
	if err != nil {
		return nil, err
	}
	if _, err := s.ConfirmMovement(ctx, mov2.ID, actor); err != nil {
		return nil, err
	}
	return s.repo.FindMovementByID(ctx, mov2.ID)
}

// stockCardID generates a deterministic ID for a stock card from item+warehouse.
func stockCardID(itemCode, warehouseCode string) string {
	return core.RowID("stock_card", itemCode, warehouseCode)
}

// movementCodePrefix returns the code prefix for a movement type.
func movementCodePrefix(mt inventory.MovementType) string {
	switch mt {
	case inventory.MovementReceipt:
		return "PN"
	case inventory.MovementDispatch:
		return "PX"
	case inventory.MovementTransferIn:
		return "PCCD"
	case inventory.MovementTransferOut:
		return "PCCT"
	case inventory.MovementAdjustmentPlus:
		return "DK"
	case inventory.MovementAdjustmentMinus:
		return "DK"
	case inventory.MovementOpeningBalance:
		return "DK"
	default:
		return "MK"
	}
}
