package inventory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appinventory "goGL/internal/application/inventory"
	domaininventory "goGL/internal/domain/inventory"
	"goGL/internal/infrastructure/db"
	persinventory "goGL/internal/infrastructure/persistence/inventory"
)

func setupTestHandler(t *testing.T) (*sql.DB, *gin.Engine) {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	repo := persinventory.NewSqliteRepository(d)
	svc := appinventory.NewService(repo)
	h := NewHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	h.Register(v1)
	return d, r
}

func TestCreateItem(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Steel Rods","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var item domaininventory.Item
	if err := json.NewDecoder(w.Body).Decode(&item); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if item.Code == "" {
		t.Error("expected auto-generated code")
	}
	if item.Name != "Steel Rods" {
		t.Errorf("name: got %q, want 'Steel Rods'", item.Name)
	}
}

func TestCreateItem_BadRequest(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"unit":"kg"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetItem(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Steel Rods","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Item
	json.NewDecoder(w.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/items/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestGetItem_NotFound(t *testing.T) {
	_, r := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/items/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestGetItemByCode(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Steel Rods","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Item
	json.NewDecoder(w.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/items/code/"+created.Code, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestUpdateItem(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Steel Rods","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Item
	json.NewDecoder(w.Body).Decode(&created)

	updateBody := `{"name":"Updated Name"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/items/"+created.ID, bytes.NewBufferString(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var updated domaininventory.Item
	json.NewDecoder(w2.Body).Decode(&updated)
	if updated.Name != "Updated Name" {
		t.Errorf("name: got %q, want 'Updated Name'", updated.Name)
	}
}

func TestDeactivateItem(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Steel Rods","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Item
	json.NewDecoder(w.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/inventory/items/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusNoContent, w2.Body.String())
	}
}

func TestListItems(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Steel Rods","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/items", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestCreateWarehouse(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Main Warehouse","warehouse_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var wh domaininventory.Warehouse
	if err := json.NewDecoder(w.Body).Decode(&wh); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if wh.Code == "" {
		t.Error("expected auto-generated code")
	}
}

func TestGetWarehouse(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Main Warehouse","warehouse_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Warehouse
	json.NewDecoder(w.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/warehouses/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestUpdateWarehouse(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Main Warehouse","warehouse_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Warehouse
	json.NewDecoder(w.Body).Decode(&created)

	updateBody := `{"name":"Updated Warehouse"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/inventory/warehouses/"+created.ID, bytes.NewBufferString(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var updated domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&updated)
	if updated.Name != "Updated Warehouse" {
		t.Errorf("name: got %q, want 'Updated Warehouse'", updated.Name)
	}
}

func TestDeactivateWarehouse(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Main Warehouse","warehouse_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created domaininventory.Warehouse
	json.NewDecoder(w.Body).Decode(&created)

	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/inventory/warehouses/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusNoContent, w2.Body.String())
	}
}

func TestListWarehouses(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"name":"Main Warehouse","warehouse_type":"general"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/warehouses", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestGetStockBalance(t *testing.T) {
	_, r := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/MH-00001/KHO-001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// --- Movement handler tests ---

func TestCreateMovement(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"movement_type":"receipt","item_code":"MH-00001","warehouse_code":"KHO-001","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "test-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "PN-") {
		t.Errorf("expected movement code starting with PN-, got %s", w.Body.String())
	}
}

func TestCreateMovement_InvalidType(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"movement_type":"bad","item_code":"MH-00001","warehouse_code":"KHO-001","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetMovement(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)
	运动 := &domaininventory.StockMovement{
		ID:            "test-mv-1",
		MovementCode:  "PN-00001",
		MovementType:  domaininventory.MovementReceipt,
		MovementDate:  "2026-09-01",
		ItemCode:      "MH-00001",
		WarehouseCode: "KHO-001",
		Quantity:      100,
		UnitPrice:     50000,
		TotalCost:     5000000,
		Status:        domaininventory.MovementDraft,
		CreatedBy:     "test-user",
		CreatedAt:     now,
	}
	_ = repo.CreateMovement(ctx, 运动)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/movements/test-mv-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestConfirmMovement(t *testing.T) {
	d, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(d)

	运动 := &domaininventory.StockMovement{
		ID:            "test-mv-2",
		MovementCode:  "PN-00002",
		MovementType:  domaininventory.MovementReceipt,
		MovementDate:  "2026-09-01",
		ItemCode:      "MH-00001",
		WarehouseCode: "KHO-001",
		Quantity:      100,
		UnitPrice:     50000,
		TotalCost:     5000000,
		Status:        domaininventory.MovementDraft,
		CreatedBy:     "test-user",
		CreatedAt:     now,
	}
	_ = repo.CreateMovement(ctx, 运动)

	item := &domaininventory.Item{
		ID:              "test-item-1",
		Code:            "MH-00001",
		Name:            "Steel",
		Category:        domaininventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: domaininventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          domaininventory.ItemActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.CreateItem(ctx, item)

	wh := &domaininventory.Warehouse{
		ID:            "test-wh-1",
		Code:          "KHO-001",
		Name:          "Main",
		WarehouseType: domaininventory.WarehouseTypeGeneral,
		Status:        domaininventory.WarehouseActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = repo.CreateWarehouse(ctx, wh)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/test-mv-2/confirm", nil)
	req.Header.Set("X-User-Id", "confirmer")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "confirmed") {
		t.Errorf("expected confirmed status, got %s", w.Body.String())
	}
}

func TestListMovements(t *testing.T) {
	_, r := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/movements?item=MH-00001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestTransferStock(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	item := &domaininventory.Item{
		ID:              "test-item-t",
		Code:            "MH-00100",
		Name:            "Transfer Item",
		Category:        domaininventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: domaininventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          domaininventory.ItemActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.CreateItem(ctx, item)

	for _, code := range []string{"KHO-100", "KHO-101"} {
		wh := &domaininventory.Warehouse{
			ID:            "test-wh-" + code,
			Code:          code,
			Name:          "Warehouse " + code,
			WarehouseType: domaininventory.WarehouseTypeGeneral,
			Status:        domaininventory.WarehouseActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		_ = repo.CreateWarehouse(ctx, wh)
	}

	// Create and confirm a receipt first to have stock
	receiptBody := `{"movement_type":"receipt","item_code":"MH-00100","warehouse_code":"KHO-100","quantity":200,"unit_price":50000,"movement_date":"2026-09-01"}`
	receiptReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(receiptBody))
	receiptReq.Header.Set("Content-Type", "application/json")
	receiptReq.Header.Set("X-User-Id", "test-user")
	receiptRec := httptest.NewRecorder()
	r.ServeHTTP(receiptRec, receiptReq)

	if receiptRec.Code != http.StatusCreated {
		t.Fatalf("receipt create: got %d, want %d; body: %s", receiptRec.Code, http.StatusCreated, receiptRec.Body.String())
	}

	// Extract movement ID from response
	var receiptResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(receiptRec.Body.Bytes(), &receiptResp)

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+receiptResp.ID+"/confirm", nil)
	confirmReq.Header.Set("X-User-Id", "test-user")
	confirmRec := httptest.NewRecorder()
	r.ServeHTTP(confirmRec, confirmReq)

	if confirmRec.Code != http.StatusOK {
		t.Fatalf("receipt confirm: got %d, want %d; body: %s", confirmRec.Code, http.StatusOK, confirmRec.Body.String())
	}

	// Now test transfer
	body := `{"item_code":"MH-00100","from_warehouse":"KHO-100","to_warehouse":"KHO-101","quantity":50,"unit_price":50000,"movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfer", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "transfer-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "outbound") || !strings.Contains(w.Body.String(), "inbound") {
		t.Errorf("expected outbound and inbound in response, got %s", w.Body.String())
	}
}

func TestTransferStock_SameWarehouse(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"item_code":"MH-00001","from_warehouse":"KHO-001","to_warehouse":"KHO-001","quantity":50,"unit_price":50000,"movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/transfer", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAdjustStock(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	item := &domaininventory.Item{
		ID:              "test-item-adj",
		Code:            "MH-00200",
		Name:            "Adjust Item",
		Category:        domaininventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: domaininventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          domaininventory.ItemActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.CreateItem(ctx, item)

	wh := &domaininventory.Warehouse{
		ID:            "test-wh-adj",
		Code:          "KHO-200",
		Name:          "Adjust Warehouse",
		WarehouseType: domaininventory.WarehouseTypeGeneral,
		Status:        domaininventory.WarehouseActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = repo.CreateWarehouse(ctx, wh)

	body := `{"item_code":"MH-00200","warehouse_code":"KHO-200","quantity":10,"unit_price":50000,"reason":"Count adjustment","movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/adjust", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "adjust-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "adjustment_plus") {
		t.Errorf("expected adjustment_plus type, got %s", w.Body.String())
	}
}

func TestAdjustStock_NoReason(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"item_code":"MH-00001","warehouse_code":"KHO-001","quantity":10,"unit_price":50000,"reason":"","movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/adjust", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestAdjustStock_NegativeQuantity(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	item := &domaininventory.Item{
		ID:              "test-item-adj2",
		Code:            "MH-00201",
		Name:            "Adjust Item 2",
		Category:        domaininventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: domaininventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          domaininventory.ItemActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.CreateItem(ctx, item)

	wh := &domaininventory.Warehouse{
		ID:            "test-wh-adj2",
		Code:          "KHO-201",
		Name:          "Adjust Warehouse 2",
		WarehouseType: domaininventory.WarehouseTypeGeneral,
		Status:        domaininventory.WarehouseActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = repo.CreateWarehouse(ctx, wh)

	// First receive some stock
	receiptBody := `{"movement_type":"receipt","item_code":"MH-00201","warehouse_code":"KHO-201","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`
	receiptReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(receiptBody))
	receiptReq.Header.Set("Content-Type", "application/json")
	receiptReq.Header.Set("X-User-Id", "test-user")
	receiptRec := httptest.NewRecorder()
	r.ServeHTTP(receiptRec, receiptReq)

	if receiptRec.Code != http.StatusCreated {
		t.Fatalf("receipt create: got %d, want %d; body: %s", receiptRec.Code, http.StatusCreated, receiptRec.Body.String())
	}

	var receiptResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(receiptRec.Body.Bytes(), &receiptResp)

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+receiptResp.ID+"/confirm", nil)
	confirmReq.Header.Set("X-User-Id", "test-user")
	confirmRec := httptest.NewRecorder()
	r.ServeHTTP(confirmRec, confirmReq)

	if confirmRec.Code != http.StatusOK {
		t.Fatalf("receipt confirm: got %d, want %d; body: %s", confirmRec.Code, http.StatusOK, confirmRec.Body.String())
	}

	// Now adjust down
	body := `{"item_code":"MH-00201","warehouse_code":"KHO-201","quantity":-10,"unit_price":50000,"reason":"Damaged goods","movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/adjust", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "adjust-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "adjustment_minus") {
		t.Errorf("expected adjustment_minus type, got %s", w.Body.String())
	}
}

// --- Physical Count handler tests ---

func TestCreatePhysicalCount(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"warehouse_code":"KHO-001","count_date":"2026-09-01","lines":[{"item_code":"MH-00001","counted_qty":100}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/counts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "count-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "PC-") {
		t.Errorf("expected count code starting with PC-, got %s", w.Body.String())
	}
}

func TestCreatePhysicalCount_EmptyWarehouse(t *testing.T) {
	_, r := setupTestHandler(t)

	body := `{"count_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/counts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetPhysicalCount(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	pc := &domaininventory.PhysicalCount{
		ID:            "test-pc-1",
		CountCode:     "PC-00001",
		WarehouseCode: "KHO-001",
		CountDate:     "2026-09-01",
		Status:        domaininventory.PhysicalCountDraft,
		Lines:         []domaininventory.PhysicalCountLine{{ItemCode: "MH-00001", CountedQty: 100}},
		CreatedBy:     "test-user",
		CreatedAt:     now,
	}
	_ = repo.CreatePhysicalCount(ctx, pc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/counts/test-pc-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestListPhysicalCounts(t *testing.T) {
	_, r := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/counts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestCompletePhysicalCount(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	pc := &domaininventory.PhysicalCount{
		ID:            "test-pc-2",
		CountCode:     "PC-00002",
		WarehouseCode: "KHO-001",
		CountDate:     "2026-09-01",
		Status:        domaininventory.PhysicalCountDraft,
		Lines:         []domaininventory.PhysicalCountLine{{ItemCode: "MH-00001", CountedQty: 50}},
		CreatedBy:     "test-user",
		CreatedAt:     now,
	}
	_ = repo.CreatePhysicalCount(ctx, pc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/counts/test-pc-2/complete", nil)
	req.Header.Set("X-User-Id", "completer")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "completed") {
		t.Errorf("expected completed status, got %s", w.Body.String())
	}
}

// --- NRV Write-Down handler tests ---

func TestWriteDownNRV(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	item := &domaininventory.Item{
		ID:              "test-item-nrv",
		Code:            "MH-00300",
		Name:            "NRV Item",
		Category:        domaininventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: domaininventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          domaininventory.ItemActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.CreateItem(ctx, item)

	wh := &domaininventory.Warehouse{
		ID:            "test-wh-nrv",
		Code:          "KHO-300",
		Name:          "NRV Warehouse",
		WarehouseType: domaininventory.WarehouseTypeGeneral,
		Status:        domaininventory.WarehouseActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = repo.CreateWarehouse(ctx, wh)

	// First receive some stock
	receiptBody := `{"movement_type":"receipt","item_code":"MH-00300","warehouse_code":"KHO-300","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`
	receiptReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(receiptBody))
	receiptReq.Header.Set("Content-Type", "application/json")
	receiptReq.Header.Set("X-User-Id", "test-user")
	receiptRec := httptest.NewRecorder()
	r.ServeHTTP(receiptRec, receiptReq)

	if receiptRec.Code != http.StatusCreated {
		t.Fatalf("receipt create: got %d, want %d; body: %s", receiptRec.Code, http.StatusCreated, receiptRec.Body.String())
	}

	var receiptResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(receiptRec.Body.Bytes(), &receiptResp)

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+receiptResp.ID+"/confirm", nil)
	confirmReq.Header.Set("X-User-Id", "test-user")
	confirmRec := httptest.NewRecorder()
	r.ServeHTTP(confirmRec, confirmReq)

	if confirmRec.Code != http.StatusOK {
		t.Fatalf("receipt confirm: got %d, want %d; body: %s", confirmRec.Code, http.StatusOK, confirmRec.Body.String())
	}

	// Now write down NRV (NRV is 40000, cost is 50000)
	body := `{"item_code":"MH-00300","warehouse_code":"KHO-300","nrv_unit_cost":40000,"movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/writedown", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "nrv-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "nrv_write_down") {
		t.Errorf("expected nrv_write_down reference, got %s", w.Body.String())
	}
}

func TestWriteDownNRV_NoWriteDownNeeded(t *testing.T) {
	db, r := setupTestHandler(t)

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)
	repo := persinventory.NewSqliteRepository(db)

	item := &domaininventory.Item{
		ID:              "test-item-nrv2",
		Code:            "MH-00301",
		Name:            "NRV Item 2",
		Category:        domaininventory.CategoryRawMaterials,
		Unit:            "kg",
		ValuationMethod: domaininventory.ValuationFIFO,
		GLAccount152:    "1521",
		GLAccount632:    "6321",
		Status:          domaininventory.ItemActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.CreateItem(ctx, item)

	wh := &domaininventory.Warehouse{
		ID:            "test-wh-nrv2",
		Code:          "KHO-301",
		Name:          "NRV Warehouse 2",
		WarehouseType: domaininventory.WarehouseTypeGeneral,
		Status:        domaininventory.WarehouseActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = repo.CreateWarehouse(ctx, wh)

	// First receive some stock
	receiptBody := `{"movement_type":"receipt","item_code":"MH-00301","warehouse_code":"KHO-301","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`
	receiptReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(receiptBody))
	receiptReq.Header.Set("Content-Type", "application/json")
	receiptReq.Header.Set("X-User-Id", "test-user")
	receiptRec := httptest.NewRecorder()
	r.ServeHTTP(receiptRec, receiptReq)

	if receiptRec.Code != http.StatusCreated {
		t.Fatalf("receipt create: got %d, want %d; body: %s", receiptRec.Code, http.StatusCreated, receiptRec.Body.String())
	}

	var receiptResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(receiptRec.Body.Bytes(), &receiptResp)

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+receiptResp.ID+"/confirm", nil)
	confirmReq.Header.Set("X-User-Id", "test-user")
	confirmRec := httptest.NewRecorder()
	r.ServeHTTP(confirmRec, confirmReq)

	if confirmRec.Code != http.StatusOK {
		t.Fatalf("receipt confirm: got %d, want %d; body: %s", confirmRec.Code, http.StatusOK, confirmRec.Body.String())
	}

	// NRV is higher than cost - no write-down needed
	body := `{"item_code":"MH-00301","warehouse_code":"KHO-301","nrv_unit_cost":60000,"movement_date":"2026-09-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/writedown", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// --- Report handler tests ---

func TestGetStockBalanceReport(t *testing.T) {
	_, r := setupTestHandler(t)

	// Create item and warehouse
	itemBody := `{"name":"Report Item","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(itemBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var item domaininventory.Item
	json.NewDecoder(w.Body).Decode(&item)

	whBody := `{"name":"Report Warehouse","warehouse_type":"general"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(whBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var wh domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&wh)

	// Create receipt movement
	movBody := fmt.Sprintf(`{"movement_type":"receipt","item_code":"%s","warehouse_code":"%s","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`, item.Code, wh.Code)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(movBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var mov domaininventory.StockMovement
	json.NewDecoder(w3.Body).Decode(&mov)

	// Confirm movement
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+mov.ID+"/confirm", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	// Get stock balance report
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/reports/balance", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w5.Code, http.StatusOK, w5.Body.String())
	}

	var report []domaininventory.StockCard
	if err := json.NewDecoder(w5.Body).Decode(&report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("expected 1 stock card, got %d", len(report))
	}
	if int64(report[0].CurrentQty) != 100 {
		t.Errorf("quantity: got %v, want 100", report[0].CurrentQty)
	}
}

func TestGetStockMovementReport(t *testing.T) {
	_, r := setupTestHandler(t)

	// Create item and warehouse
	itemBody := `{"name":"Movement Report Item","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(itemBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var item domaininventory.Item
	json.NewDecoder(w.Body).Decode(&item)

	whBody := `{"name":"Movement Report Warehouse","warehouse_type":"general"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(whBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var wh domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&wh)

	// Create and confirm receipt movement
	movBody := fmt.Sprintf(`{"movement_type":"receipt","item_code":"%s","warehouse_code":"%s","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`, item.Code, wh.Code)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(movBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var mov domaininventory.StockMovement
	json.NewDecoder(w3.Body).Decode(&mov)

	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+mov.ID+"/confirm", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	// Get stock movement report
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/reports/movements", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w5.Code, http.StatusOK, w5.Body.String())
	}

	var report []domaininventory.StockMovement
	if err := json.NewDecoder(w5.Body).Decode(&report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("expected 1 movement, got %d", len(report))
	}
}

func TestGetStockValuationReport(t *testing.T) {
	_, r := setupTestHandler(t)

	// Create item and warehouse
	itemBody := `{"name":"Valuation Report Item","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(itemBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var item domaininventory.Item
	json.NewDecoder(w.Body).Decode(&item)

	whBody := `{"name":"Valuation Report Warehouse","warehouse_type":"general"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(whBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var wh domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&wh)

	// Create and confirm receipt movement
	movBody := fmt.Sprintf(`{"movement_type":"receipt","item_code":"%s","warehouse_code":"%s","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`, item.Code, wh.Code)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(movBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var mov domaininventory.StockMovement
	json.NewDecoder(w3.Body).Decode(&mov)

	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+mov.ID+"/confirm", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	// Get stock valuation report
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/reports/valuation", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w5.Code, http.StatusOK, w5.Body.String())
	}

	var report []domaininventory.StockCard
	if err := json.NewDecoder(w5.Body).Decode(&report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("expected 1 stock card, got %d", len(report))
	}
	if report[0].CurrentValue != 5000000 {
		t.Errorf("current value: got %d, want 5000000", report[0].CurrentValue)
	}
}

func TestImportOpeningBalance(t *testing.T) {
	_, r := setupTestHandler(t)

	// Create item and warehouse
	itemBody := `{"name":"Opening Balance Item","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(itemBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var item domaininventory.Item
	json.NewDecoder(w.Body).Decode(&item)

	whBody := `{"name":"Opening Balance Warehouse","warehouse_type":"general"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(whBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var wh domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&wh)

	// Import opening balance
	importBody := fmt.Sprintf(`{"item_code":"%s","warehouse_code":"%s","quantity":200,"unit_price":45000,"movement_date":"2026-01-01"}`, item.Code, wh.Code)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/opening-balance", bytes.NewBufferString(importBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w3.Code, http.StatusOK, w3.Body.String())
	}

	// Verify stock card was created
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/"+item.Code+"/"+wh.Code, nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w4.Code, http.StatusOK, w4.Body.String())
	}

	var sc domaininventory.StockCard
	json.NewDecoder(w4.Body).Decode(&sc)
	if int64(sc.CurrentQty) != 200 {
		t.Errorf("quantity: got %v, want 200", sc.CurrentQty)
	}
	if int64(sc.AverageCost) != 45000 {
		t.Errorf("average cost: got %v, want 45000", sc.AverageCost)
	}
}

func TestCreateReceiptMovement(t *testing.T) {
	_, r := setupTestHandler(t)

	// Create item and warehouse
	itemBody := `{"name":"Receipt Movement Item","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(itemBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var item domaininventory.Item
	json.NewDecoder(w.Body).Decode(&item)

	whBody := `{"name":"Receipt Movement Warehouse","warehouse_type":"general"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(whBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var wh domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&wh)

	// Create receipt movement via API (simulating purchase integration)
	movBody := fmt.Sprintf(`{"movement_type":"receipt","item_code":"%s","warehouse_code":"%s","quantity":50,"unit_price":60000,"movement_date":"2026-09-01","ref_doc_type":"goods_receipt","ref_doc_id":"GR-001","ref_doc_no":"NK-00001"}`, item.Code, wh.Code)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(movBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d; body: %s", w3.Code, http.StatusCreated, w3.Body.String())
	}

	var mov domaininventory.StockMovement
	json.NewDecoder(w3.Body).Decode(&mov)

	// Confirm movement
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+mov.ID+"/confirm", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w4.Code, http.StatusOK, w4.Body.String())
	}

	// Verify stock card was created
	req5 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/"+item.Code+"/"+wh.Code, nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)

	if w5.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w5.Code, http.StatusOK, w5.Body.String())
	}

	var sc domaininventory.StockCard
	json.NewDecoder(w5.Body).Decode(&sc)
	if int64(sc.CurrentQty) != 50 {
		t.Errorf("quantity: got %v, want 50", sc.CurrentQty)
	}
}

func TestCreateDispatchMovement(t *testing.T) {
	_, r := setupTestHandler(t)

	// Create item and warehouse
	itemBody := `{"name":"Dispatch Movement Item","unit":"kg","category":"raw_materials","gl_account_152":"1521"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/items", bytes.NewBufferString(itemBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var item domaininventory.Item
	json.NewDecoder(w.Body).Decode(&item)

	whBody := `{"name":"Dispatch Movement Warehouse","warehouse_type":"general"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/warehouses", bytes.NewBufferString(whBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var wh domaininventory.Warehouse
	json.NewDecoder(w2.Body).Decode(&wh)

	// First create and confirm a receipt to have stock
	receiptBody := fmt.Sprintf(`{"movement_type":"receipt","item_code":"%s","warehouse_code":"%s","quantity":100,"unit_price":50000,"movement_date":"2026-09-01"}`, item.Code, wh.Code)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(receiptBody))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var receiptMov domaininventory.StockMovement
	json.NewDecoder(w3.Body).Decode(&receiptMov)

	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+receiptMov.ID+"/confirm", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)

	// Create dispatch movement via API (simulating sales integration)
	dispatchBody := fmt.Sprintf(`{"movement_type":"dispatch","item_code":"%s","warehouse_code":"%s","quantity":30,"unit_price":70000,"movement_date":"2026-09-02","ref_doc_type":"sales_invoice","ref_doc_id":"SI-001","ref_doc_no":"PX-00001"}`, item.Code, wh.Code)
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements", bytes.NewBufferString(dispatchBody))
	req5.Header.Set("Content-Type", "application/json")
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)

	if w5.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d; body: %s", w5.Code, http.StatusCreated, w5.Body.String())
	}

	var dispatchMov domaininventory.StockMovement
	json.NewDecoder(w5.Body).Decode(&dispatchMov)

	// Confirm dispatch movement
	req6 := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/"+dispatchMov.ID+"/confirm", nil)
	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, req6)

	if w6.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w6.Code, http.StatusOK, w6.Body.String())
	}

	// Verify stock card was updated
	req7 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock/"+item.Code+"/"+wh.Code, nil)
	w7 := httptest.NewRecorder()
	r.ServeHTTP(w7, req7)

	if w7.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d; body: %s", w7.Code, http.StatusOK, w7.Body.String())
	}

	var sc domaininventory.StockCard
	json.NewDecoder(w7.Body).Decode(&sc)
	if int64(sc.CurrentQty) != 70 {
		t.Errorf("quantity: got %v, want 70", sc.CurrentQty)
	}
}
