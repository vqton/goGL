package inventory

import (
	"context"
	"database/sql"
	"testing"

	"goGL/internal/domain/inventory"
	"goGL/internal/infrastructure/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return d
}

func TestCreateAndFindItem(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	item := &inventory.Item{
		Name:            "Steel Rods",
		Category:        inventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: inventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          inventory.ItemActive,
	}
	if err := r.CreateItem(ctx, item); err != nil {
		t.Fatalf("create item: %v", err)
	}

	found, err := r.FindItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("find item: %v", err)
	}
	if found.Name != item.Name {
		t.Errorf("name: got %q, want %q", found.Name, item.Name)
	}
	if found.Category != inventory.CategoryRawMaterials {
		t.Errorf("category: got %q, want raw_materials", found.Category)
	}
}

func TestFindItemByCode(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	item := &inventory.Item{
		ID:              "test-item-1",
		Code:            "MH-00001",
		Name:            "Steel Rods",
		Category:        inventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: inventory.ValuationFIFO,
		GLAccount152:    "1521",
		Status:          inventory.ItemActive,
	}
	if err := r.CreateItem(ctx, item); err != nil {
		t.Fatalf("create item: %v", err)
	}

	found, err := r.FindItemByCode(ctx, "MH-00001")
	if err != nil {
		t.Fatalf("find by code: %v", err)
	}
	if found.ID != item.ID {
		t.Errorf("ID: got %q, want %q", found.ID, item.ID)
	}
}

func TestFindItem_NotFound(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	_, err := r.FindItemByID(ctx, "nonexistent")
	if err != inventory.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateItem(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	item := &inventory.Item{
		ID:              "test-item-2",
		Name:            "Original Name",
		Category:        inventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: inventory.ValuationFIFO,
		GLAccount152:    "1521",
		Status:          inventory.ItemActive,
	}
	if err := r.CreateItem(ctx, item); err != nil {
		t.Fatalf("create item: %v", err)
	}

	item.Name = "Updated Name"
	if err := r.UpdateItem(ctx, item); err != nil {
		t.Fatalf("update item: %v", err)
	}

	found, err := r.FindItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("find item: %v", err)
	}
	if found.Name != "Updated Name" {
		t.Errorf("name: got %q, want 'Updated Name'", found.Name)
	}
}

func TestDeleteItem(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	item := &inventory.Item{ID: "test-item-3", Name: "To Delete", Category: inventory.CategoryRawMaterials, Unit: "pcs", ValuationMethod: inventory.ValuationFIFO, GLAccount152: "1521", Status: inventory.ItemActive}
	if err := r.CreateItem(ctx, item); err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := r.DeleteItem(ctx, item.ID); err != nil {
		t.Fatalf("delete item: %v", err)
	}
	_, err := r.FindItemByID(ctx, item.ID)
	if err != inventory.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestListItems(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		item := &inventory.Item{
			ID:              "item-" + string(rune('0'+i)),
			Name:            "Item",
			Category:        inventory.CategoryRawMaterials,
			Unit:            "pcs",
			ValuationMethod: inventory.ValuationFIFO,
			GLAccount152:    "1521",
			Status:          inventory.ItemActive,
		}
		r.CreateItem(ctx, item)
	}

	items, total, err := r.ListItems(ctx, "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if total != 5 {
		t.Errorf("total: got %d, want 5", total)
	}
	if len(items) != 5 {
		t.Errorf("len: got %d, want 5", len(items))
	}
}

func TestNextItemCode(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	n1, err := r.NextItemCode(ctx)
	if err != nil {
		t.Fatalf("next code: %v", err)
	}
	n2, err := r.NextItemCode(ctx)
	if err != nil {
		t.Fatalf("next code: %v", err)
	}
	if n2 != n1+1 {
		t.Errorf("sequence: got %d, want %d", n2, n1+1)
	}
}

func TestCreateAndFindWarehouse(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	wh := &inventory.Warehouse{
		Name:          "Main Warehouse",
		WarehouseType: inventory.WarehouseTypeGeneral,
		Status:        inventory.WarehouseActive,
	}
	if err := r.CreateWarehouse(ctx, wh); err != nil {
		t.Fatalf("create warehouse: %v", err)
	}

	found, err := r.FindWarehouseByID(ctx, wh.ID)
	if err != nil {
		t.Fatalf("find warehouse: %v", err)
	}
	if found.Name != wh.Name {
		t.Errorf("name: got %q, want %q", found.Name, wh.Name)
	}
}

func TestFindWarehouseByCode(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	wh := &inventory.Warehouse{ID: "wh-1", Code: "KHO-001", Name: "Main", WarehouseType: inventory.WarehouseTypeGeneral, Status: inventory.WarehouseActive}
	if err := r.CreateWarehouse(ctx, wh); err != nil {
		t.Fatalf("create warehouse: %v", err)
	}

	found, err := r.FindWarehouseByCode(ctx, "KHO-001")
	if err != nil {
		t.Fatalf("find by code: %v", err)
	}
	if found.ID != wh.ID {
		t.Errorf("ID: got %q, want %q", found.ID, wh.ID)
	}
}

func TestUpdateWarehouse(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	wh := &inventory.Warehouse{ID: "wh-2", Name: "Original", WarehouseType: inventory.WarehouseTypeGeneral, Status: inventory.WarehouseActive}
	if err := r.CreateWarehouse(ctx, wh); err != nil {
		t.Fatalf("create warehouse: %v", err)
	}

	wh.Name = "Updated"
	if err := r.UpdateWarehouse(ctx, wh); err != nil {
		t.Fatalf("update warehouse: %v", err)
	}

	found, err := r.FindWarehouseByID(ctx, wh.ID)
	if err != nil {
		t.Fatalf("find warehouse: %v", err)
	}
	if found.Name != "Updated" {
		t.Errorf("name: got %q, want 'Updated'", found.Name)
	}
}

func TestDeleteWarehouse(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	wh := &inventory.Warehouse{ID: "wh-3", Name: "To Delete", WarehouseType: inventory.WarehouseTypeGeneral, Status: inventory.WarehouseActive}
	if err := r.CreateWarehouse(ctx, wh); err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	if err := r.DeleteWarehouse(ctx, wh.ID); err != nil {
		t.Fatalf("delete warehouse: %v", err)
	}
	_, err := r.FindWarehouseByID(ctx, wh.ID)
	if err != inventory.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestListWarehouses(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		wh := &inventory.Warehouse{ID: "wh-" + string(rune('0'+i)), Name: "Warehouse", WarehouseType: inventory.WarehouseTypeGeneral, Status: inventory.WarehouseActive}
		r.CreateWarehouse(ctx, wh)
	}

	whs, total, err := r.ListWarehouses(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("list warehouses: %v", err)
	}
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
	if len(whs) != 3 {
		t.Errorf("len: got %d, want 3", len(whs))
	}
}

func TestNextWarehouseCode(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	n1, err := r.NextWarehouseCode(ctx)
	if err != nil {
		t.Fatalf("next code: %v", err)
	}
	n2, err := r.NextWarehouseCode(ctx)
	if err != nil {
		t.Fatalf("next code: %v", err)
	}
	if n2 != n1+1 {
		t.Errorf("sequence: got %d, want %d", n2, n1+1)
	}
}

func TestUpsertStockCard(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	sc := &inventory.StockCard{
		ID:            "card-1",
		ItemCode:      "MH-00001",
		WarehouseCode: "KHO-001",
		CurrentQty:    100,
		CurrentValue:  500000,
	}
	if err := r.UpsertStockCard(ctx, sc); err != nil {
		t.Fatalf("upsert stock card: %v", err)
	}

	found, err := r.GetStockCard(ctx, "MH-00001", "KHO-001")
	if err != nil {
		t.Fatalf("get stock card: %v", err)
	}
	if found.CurrentQty != 100 {
		t.Errorf("qty: got %f, want 100", found.CurrentQty)
	}

	// Upsert again with different qty — should update, not create duplicate
	sc2 := &inventory.StockCard{
		ID:            "card-1",
		ItemCode:      "MH-00001",
		WarehouseCode: "KHO-001",
		CurrentQty:    200,
		CurrentValue:  1000000,
	}
	if err := r.UpsertStockCard(ctx, sc2); err != nil {
		t.Fatalf("upsert stock card again: %v", err)
	}
	found2, err := r.GetStockCard(ctx, "MH-00001", "KHO-001")
	if err != nil {
		t.Fatalf("get stock card: %v", err)
	}
	if found2.CurrentQty != 200 {
		t.Errorf("qty after upsert: got %f, want 200", found2.CurrentQty)
	}
}

func TestListStockCards(t *testing.T) {
	d := setupTestDB(t)
	defer d.Close()
	r := NewSqliteRepository(d)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		sc := &inventory.StockCard{
			ID:            "card-" + string(rune('0'+i)),
			ItemCode:      "MH-00001",
			WarehouseCode: "KHO-001",
			CurrentQty:    float64(i * 10),
		}
		r.UpsertStockCard(ctx, sc)
	}

	cards, total, err := r.ListStockCards(ctx, "KHO-001", 10, 0)
	if err != nil {
		t.Fatalf("list stock cards: %v", err)
	}
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
	if len(cards) != 3 {
		t.Errorf("len: got %d, want 3", len(cards))
	}
}
