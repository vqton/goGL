package inventory

import (
	"errors"
	"testing"

	"goGL/internal/domain/core"
)

// --- Item validation tests ---

func TestValidateItem_Valid(t *testing.T) {
	i := &Item{Name: "Steel Rods", Unit: "kg", Category: CategoryRawMaterials, GLAccount152: "1521"}
	if err := ValidateItem(i); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.ValuationMethod != ValuationFIFO {
		t.Errorf("expected FIFO default, got %s", i.ValuationMethod)
	}
	if i.Status != ItemActive {
		t.Errorf("expected active default, got %s", i.Status)
	}
}

func TestValidateItem_EmptyName(t *testing.T) {
	i := &Item{Unit: "kg", Category: CategoryRawMaterials, GLAccount152: "1521"}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	var ve *core.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Field != "name" {
		t.Errorf("expected field 'name', got %s", ve.Field)
	}
}

func TestValidateItem_EmptyUnit(t *testing.T) {
	i := &Item{Name: "Steel", Category: CategoryRawMaterials, GLAccount152: "1521"}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for empty unit")
	}
}

func TestValidateItem_EmptyCategory(t *testing.T) {
	i := &Item{Name: "Steel", Unit: "kg", GLAccount152: "1521"}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for empty category")
	}
}

func TestValidateItem_InvalidCategory(t *testing.T) {
	i := &Item{Name: "Steel", Unit: "kg", Category: "bad", GLAccount152: "1521"}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestValidateItem_InvalidValuationMethod(t *testing.T) {
	i := &Item{Name: "Steel", Unit: "kg", Category: CategoryRawMaterials, GLAccount152: "1521", ValuationMethod: "lifo"}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for LIFO (prohibited)")
	}
}

func TestValidateItem_EmptyGLAccount(t *testing.T) {
	i := &Item{Name: "Steel", Unit: "kg", Category: CategoryRawMaterials}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for empty GL account")
	}
}

func TestValidateItem_InvalidStatus(t *testing.T) {
	i := &Item{Name: "Steel", Unit: "kg", Category: CategoryRawMaterials, GLAccount152: "1521", Status: "bad"}
	err := ValidateItem(i)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidateItem_AllCategories(t *testing.T) {
	categories := []ItemCategory{CategoryRawMaterials, CategorySupplies, CategoryFinishedGoods, CategoryWIP, CategoryConsignment}
	for _, cat := range categories {
		i := &Item{Name: "Test", Unit: "pcs", Category: cat, GLAccount152: "1521"}
		if err := ValidateItem(i); err != nil {
			t.Errorf("category %s should be valid: %v", cat, err)
		}
	}
}

func TestValidateItem_AllValuationMethods(t *testing.T) {
	methods := []ValuationMethod{ValuationFIFO, ValuationWeightedAverage}
	for _, m := range methods {
		i := &Item{Name: "Test", Unit: "pcs", Category: CategoryRawMaterials, GLAccount152: "1521", ValuationMethod: m}
		if err := ValidateItem(i); err != nil {
			t.Errorf("valuation method %s should be valid: %v", m, err)
		}
	}
}

func TestValidateItem_AllStatuses(t *testing.T) {
	statuses := []ItemStatus{ItemActive, ItemInactive, ItemDiscontinued}
	for _, s := range statuses {
		i := &Item{Name: "Test", Unit: "pcs", Category: CategoryRawMaterials, GLAccount152: "1521", Status: s}
		if err := ValidateItem(i); err != nil {
			t.Errorf("status %s should be valid: %v", s, err)
		}
	}
}

func TestItem_Clone(t *testing.T) {
	i := &Item{ID: "1", Code: "MH-00001", Name: "Steel", Category: CategoryRawMaterials}
	cp := i.Clone()
	cp.Name = "Changed"
	if i.Name == cp.Name {
		t.Error("clone should not share memory")
	}
}

func TestItemCategory_IsValid(t *testing.T) {
	if !CategoryRawMaterials.IsValid() {
		t.Error("raw_materials should be valid")
	}
	if !CategoryFinishedGoods.IsValid() {
		t.Error("finished_goods should be valid")
	}
	if ItemCategory("bad").IsValid() {
		t.Error("bad should be invalid")
	}
}

func TestValuationMethod_IsValid(t *testing.T) {
	if !ValuationFIFO.IsValid() {
		t.Error("fifo should be valid")
	}
	if !ValuationWeightedAverage.IsValid() {
		t.Error("weighted_average should be valid")
	}
	if ValuationMethod("lifo").IsValid() {
		t.Error("lifo should be invalid (prohibited)")
	}
}

func TestItemStatus_IsValid(t *testing.T) {
	if !ItemActive.IsValid() {
		t.Error("active should be valid")
	}
	if !ItemInactive.IsValid() {
		t.Error("inactive should be valid")
	}
	if !ItemDiscontinued.IsValid() {
		t.Error("discontinued should be valid")
	}
	if ItemStatus("bad").IsValid() {
		t.Error("bad should be invalid")
	}
}

// --- Warehouse validation tests ---

func TestValidateWarehouse_Valid(t *testing.T) {
	w := &Warehouse{Name: "Main Warehouse"}
	if err := ValidateWarehouse(w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.WarehouseType != WarehouseTypeGeneral {
		t.Errorf("expected general default, got %s", w.WarehouseType)
	}
	if w.Status != WarehouseActive {
		t.Errorf("expected active default, got %s", w.Status)
	}
}

func TestValidateWarehouse_EmptyName(t *testing.T) {
	w := &Warehouse{}
	err := ValidateWarehouse(w)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateWarehouse_InvalidType(t *testing.T) {
	w := &Warehouse{Name: "Test", WarehouseType: "bad"}
	err := ValidateWarehouse(w)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestValidateWarehouse_InvalidStatus(t *testing.T) {
	w := &Warehouse{Name: "Test", Status: "bad"}
	err := ValidateWarehouse(w)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidateWarehouse_AllTypes(t *testing.T) {
	types := []WarehouseType{WarehouseTypeRawMaterial, WarehouseTypeFinished, WarehouseTypeGeneral}
	for _, wt := range types {
		w := &Warehouse{Name: "Test", WarehouseType: wt}
		if err := ValidateWarehouse(w); err != nil {
			t.Errorf("type %s should be valid: %v", wt, err)
		}
	}
}

func TestValidateWarehouse_AllStatuses(t *testing.T) {
	statuses := []WarehouseStatus{WarehouseActive, WarehouseInactive}
	for _, s := range statuses {
		w := &Warehouse{Name: "Test", Status: s}
		if err := ValidateWarehouse(w); err != nil {
			t.Errorf("status %s should be valid: %v", s, err)
		}
	}
}

func TestWarehouse_Clone(t *testing.T) {
	w := &Warehouse{ID: "1", Code: "KHO-001", Name: "Main"}
	cp := w.Clone()
	cp.Name = "Changed"
	if w.Name == cp.Name {
		t.Error("clone should not share memory")
	}
}

func TestWarehouseType_IsValid(t *testing.T) {
	if !WarehouseTypeRawMaterial.IsValid() {
		t.Error("raw_material should be valid")
	}
	if !WarehouseTypeFinished.IsValid() {
		t.Error("finished_goods should be valid")
	}
	if !WarehouseTypeGeneral.IsValid() {
		t.Error("general should be valid")
	}
	if WarehouseType("bad").IsValid() {
		t.Error("bad should be invalid")
	}
}

func TestWarehouseStatus_IsValid(t *testing.T) {
	if !WarehouseActive.IsValid() {
		t.Error("active should be valid")
	}
	if !WarehouseInactive.IsValid() {
		t.Error("inactive should be valid")
	}
	if WarehouseStatus("bad").IsValid() {
		t.Error("bad should be invalid")
	}
}

// --- StockMovement validation tests ---

func TestValidateStockMovement_Valid(t *testing.T) {
	m := &StockMovement{
		MovementType:  MovementReceipt,
		ItemCode:      "MH-00001",
		WarehouseCode: "KHO-001",
		Quantity:      100,
		UnitPrice:     50000,
		MovementDate:  "2026-09-01",
	}
	if err := ValidateStockMovement(m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Status != MovementDraft {
		t.Errorf("expected draft default, got %s", m.Status)
	}
}

func TestValidateStockMovement_EmptyType(t *testing.T) {
	m := &StockMovement{ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 10, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestValidateStockMovement_InvalidType(t *testing.T) {
	m := &StockMovement{MovementType: "bad", ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 10, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestValidateStockMovement_EmptyItemCode(t *testing.T) {
	m := &StockMovement{MovementType: MovementReceipt, WarehouseCode: "KHO-001", Quantity: 10, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for empty item code")
	}
}

func TestValidateStockMovement_EmptyWarehouseCode(t *testing.T) {
	m := &StockMovement{MovementType: MovementReceipt, ItemCode: "MH-00001", Quantity: 10, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for empty warehouse code")
	}
}

func TestValidateStockMovement_ZeroQuantity(t *testing.T) {
	m := &StockMovement{MovementType: MovementReceipt, ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 0, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for zero quantity")
	}
}

func TestValidateStockMovement_NegativeQuantity(t *testing.T) {
	m := &StockMovement{MovementType: MovementReceipt, ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: -5, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for negative quantity")
	}
}

func TestValidateStockMovement_NegativeUnitPrice(t *testing.T) {
	m := &StockMovement{MovementType: MovementReceipt, ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 10, UnitPrice: -1, MovementDate: "2026-09-01"}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for negative unit price")
	}
}

func TestValidateStockMovement_EmptyDate(t *testing.T) {
	m := &StockMovement{MovementType: MovementReceipt, ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 10}
	err := ValidateStockMovement(m)
	if err == nil {
		t.Fatal("expected error for empty date")
	}
}

func TestValidateStockMovement_AllTypes(t *testing.T) {
	types := []MovementType{MovementReceipt, MovementDispatch, MovementTransferIn, MovementTransferOut, MovementAdjustmentPlus, MovementAdjustmentMinus, MovementOpeningBalance}
	for _, mt := range types {
		m := &StockMovement{MovementType: mt, ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 10, MovementDate: "2026-09-01"}
		if err := ValidateStockMovement(m); err != nil {
			t.Errorf("type %s should be valid: %v", mt, err)
		}
	}
}

func TestValidateStockMovement_AllStatuses(t *testing.T) {
	statuses := []MovementStatus{MovementDraft, MovementConfirmed, MovementPosted, MovementCancelled}
	for _, s := range statuses {
		m := &StockMovement{MovementType: MovementReceipt, ItemCode: "MH-00001", WarehouseCode: "KHO-001", Quantity: 10, MovementDate: "2026-09-01", Status: s}
		if err := ValidateStockMovement(m); err != nil {
			t.Errorf("status %s should be valid: %v", s, err)
		}
	}
}

func TestMovementType_IsInbound(t *testing.T) {
	inbound := []MovementType{MovementReceipt, MovementTransferIn, MovementAdjustmentPlus, MovementOpeningBalance}
	for _, mt := range inbound {
		if !mt.IsInbound() {
			t.Errorf("%s should be inbound", mt)
		}
	}
	outbound := []MovementType{MovementDispatch, MovementTransferOut, MovementAdjustmentMinus}
	for _, mt := range outbound {
		if mt.IsInbound() {
			t.Errorf("%s should not be inbound", mt)
		}
	}
}

func TestMovementType_IsOutbound(t *testing.T) {
	outbound := []MovementType{MovementDispatch, MovementTransferOut, MovementAdjustmentMinus}
	for _, mt := range outbound {
		if !mt.IsOutbound() {
			t.Errorf("%s should be outbound", mt)
		}
	}
	inbound := []MovementType{MovementReceipt, MovementTransferIn, MovementAdjustmentPlus, MovementOpeningBalance}
	for _, mt := range inbound {
		if mt.IsOutbound() {
			t.Errorf("%s should not be outbound", mt)
		}
	}
}

func TestStockMovement_Clone(t *testing.T) {
	m := &StockMovement{ID: "1", MovementCode: "PN-00001", ItemCode: "MH-00001"}
	cp := m.Clone()
	cp.ItemCode = "MH-00002"
	if m.ItemCode == cp.ItemCode {
		t.Error("clone should not share memory")
	}
}

// --- PhysicalCount validation tests ---

func TestValidatePhysicalCount_Valid(t *testing.T) {
	pc := &PhysicalCount{
		WarehouseCode: "KHO-001",
		CountDate:     "2026-09-01",
	}
	if err := ValidatePhysicalCount(pc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Status != PhysicalCountDraft {
		t.Errorf("expected draft default, got %s", pc.Status)
	}
}

func TestValidatePhysicalCount_EmptyWarehouse(t *testing.T) {
	pc := &PhysicalCount{CountDate: "2026-09-01"}
	err := ValidatePhysicalCount(pc)
	if err == nil {
		t.Fatal("expected error for empty warehouse")
	}
}

func TestValidatePhysicalCount_EmptyDate(t *testing.T) {
	pc := &PhysicalCount{WarehouseCode: "KHO-001"}
	err := ValidatePhysicalCount(pc)
	if err == nil {
		t.Fatal("expected error for empty date")
	}
}

func TestValidatePhysicalCount_InvalidStatus(t *testing.T) {
	pc := &PhysicalCount{WarehouseCode: "KHO-001", CountDate: "2026-09-01", Status: "bad"}
	err := ValidatePhysicalCount(pc)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestValidatePhysicalCount_AllStatuses(t *testing.T) {
	statuses := []PhysicalCountStatus{PhysicalCountDraft, PhysicalCountInProgress, PhysicalCountCompleted, PhysicalCountReconciled}
	for _, s := range statuses {
		pc := &PhysicalCount{WarehouseCode: "KHO-001", CountDate: "2026-09-01", Status: s}
		if err := ValidatePhysicalCount(pc); err != nil {
			t.Errorf("status %s should be valid: %v", s, err)
		}
	}
}

func TestPhysicalCountStatus_IsValid(t *testing.T) {
	if !PhysicalCountDraft.IsValid() {
		t.Error("draft should be valid")
	}
	if !PhysicalCountInProgress.IsValid() {
		t.Error("in_progress should be valid")
	}
	if !PhysicalCountCompleted.IsValid() {
		t.Error("completed should be valid")
	}
	if !PhysicalCountReconciled.IsValid() {
		t.Error("reconciled should be valid")
	}
	if PhysicalCountStatus("bad").IsValid() {
		t.Error("bad should be invalid")
	}
}

func TestPhysicalCount_Clone(t *testing.T) {
	pc := &PhysicalCount{
		ID:            "1",
		CountCode:     "PC-00001",
		WarehouseCode: "KHO-001",
		Lines: []PhysicalCountLine{
			{ID: "L1", ItemCode: "MH-001", CountedQty: 100},
		},
	}
	cp := pc.Clone()
	cp.Lines[0].CountedQty = 50
	if pc.Lines[0].CountedQty == cp.Lines[0].CountedQty {
		t.Error("clone should not share memory")
	}
}
