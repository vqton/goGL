package purchase

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	"goGL/internal/application/purchase"
	domainpurchase "goGL/internal/domain/purchase"
	perspurchase "goGL/internal/infrastructure/persistence/purchase"
)

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	tables := []string{
		"suppliers", "supplier_sequences",
		"purchase_orders", "purchase_order_sequences",
		"goods_receipts", "goods_receipt_sequences",
		"purchase_invoices", "purchase_invoice_sequences",
		"purchase_payments", "purchase_payment_sequences",
		"purchase_sequences",
	}
	for _, tbl := range tables {
		if _, err := db.Exec("CREATE TABLE " + tbl + " (id TEXT PRIMARY KEY, data TEXT NOT NULL)"); err != nil {
			t.Fatalf("failed to create %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() { db.Close() })

	svc := purchase.NewService(perspurchase.NewSqliteRepository(db))
	h := NewHandler(svc)
	r := gin.New()
	v1 := r.Group("/api/v1")
	h.Register(v1)
	return h, r
}

func performRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "test-admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Supplier API Tests ---

func TestAPI_CreateSupplier_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"name":     "ABC Company",
		"tax_code": "0123456789",
	}
	w := performRequest(r, "POST", "/api/v1/purchase/suppliers", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var got domainpurchase.Supplier
	json.Unmarshal(w.Body.Bytes(), &got)
	if got.RefNo == "" {
		t.Error("expected RefNo to be set")
	}
	if got.Status != domainpurchase.SupplierActive {
		t.Errorf("Status = %q, want active", got.Status)
	}
}

func TestAPI_CreateSupplier_BadRequest(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{"tax_code": "0123456789"}
	w := performRequest(r, "POST", "/api/v1/purchase/suppliers", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestAPI_GetSupplier_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{"name": "ABC", "tax_code": "001"}
	createW := performRequest(r, "POST", "/api/v1/purchase/suppliers", body)
	var created domainpurchase.Supplier
	json.Unmarshal(createW.Body.Bytes(), &created)

	w := performRequest(r, "GET", "/api/v1/purchase/suppliers/"+created.ID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestAPI_GetSupplier_NotFound(t *testing.T) {
	_, r := setupTestHandler(t)
	w := performRequest(r, "GET", "/api/v1/purchase/suppliers/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPI_ListSuppliers_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	performRequest(r, "POST", "/api/v1/purchase/suppliers", map[string]any{"name": "A", "tax_code": "001"})
	performRequest(r, "POST", "/api/v1/purchase/suppliers", map[string]any{"name": "B", "tax_code": "002"})
	w := performRequest(r, "GET", "/api/v1/purchase/suppliers?limit=10&offset=0", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	items := result["items"].([]any)
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}

func TestAPI_DeleteSupplier_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	createW := performRequest(r, "POST", "/api/v1/purchase/suppliers", map[string]any{"name": "ABC", "tax_code": "001"})
	var created domainpurchase.Supplier
	json.Unmarshal(createW.Body.Bytes(), &created)

	w := performRequest(r, "DELETE", "/api/v1/purchase/suppliers/"+created.ID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- Purchase Order API Tests ---

func TestAPI_CreateOrder_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code": "NCC-00001",
		"order_date":    "2026-08-30",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	}
	w := performRequest(r, "POST", "/api/v1/purchase/orders", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var got domainpurchase.PurchaseOrder
	json.Unmarshal(w.Body.Bytes(), &got)
	if got.RefNo == "" {
		t.Error("expected RefNo to be set")
	}
}

func TestAPI_ConfirmOrder_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code": "NCC-00001",
		"order_date":    "2026-08-30",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	}
	createW := performRequest(r, "POST", "/api/v1/purchase/orders", body)
	var created domainpurchase.PurchaseOrder
	json.Unmarshal(createW.Body.Bytes(), &created)

	w := performRequest(r, "POST", "/api/v1/purchase/orders/"+created.ID+"/confirm", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- Goods Receipt API Tests ---

func TestAPI_CreateReceipt_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"po_id":         "po-123",
		"supplier_code": "NCC-00001",
		"receipt_date":  "2026-09-10",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity_received": 10},
		},
	}
	w := performRequest(r, "POST", "/api/v1/purchase/receipts", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body: %s", w.Code, w.Body.String())
	}
}

// --- Purchase Invoice API Tests ---

func TestAPI_CreateInvoice_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code": "NCC-00001",
		"invoice_date":  "2026-09-10",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	}
	w := performRequest(r, "POST", "/api/v1/purchase/invoices", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body: %s", w.Code, w.Body.String())
	}
	var got domainpurchase.PurchaseInvoice
	json.Unmarshal(w.Body.Bytes(), &got)
	if got.EInvoiceStatus != domainpurchase.EInvoiceNone {
		t.Errorf("EInvoiceStatus = %q, want none", got.EInvoiceStatus)
	}
}

func TestAPI_PostInvoice_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code": "NCC-00001",
		"invoice_date":  "2026-09-10",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	}
	createW := performRequest(r, "POST", "/api/v1/purchase/invoices", body)
	var created domainpurchase.PurchaseInvoice
	json.Unmarshal(createW.Body.Bytes(), &created)

	w := performRequest(r, "POST", "/api/v1/purchase/invoices/"+created.ID+"/post", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- Payment API Tests ---

func TestAPI_CreatePayment_Success(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code":  "NCC-00001",
		"payment_date":   "2026-10-10",
		"payment_method": "bank_transfer",
		"amount":         map[string]any{"amount_minor": 5000000, "currency": "VND"},
		"applied_invoices": []map[string]any{
			{"invoice_id": "inv-1", "amount_applied": 5000000},
		},
	}
	w := performRequest(r, "POST", "/api/v1/purchase/payments", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body: %s", w.Code, w.Body.String())
	}
}

// --- Error Handling Tests ---

func TestAPI_CreateOrder_EmptyLines(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code": "NCC-00001",
		"order_date":    "2026-08-30",
		"lines":         []map[string]any{},
	}
	w := performRequest(r, "POST", "/api/v1/purchase/orders", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestAPI_ConfirmOrder_WrongStatus(t *testing.T) {
	_, r := setupTestHandler(t)
	body := map[string]any{
		"supplier_code": "NCC-00001",
		"order_date":    "2026-08-30",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	}
	createW := performRequest(r, "POST", "/api/v1/purchase/orders", body)
	var created domainpurchase.PurchaseOrder
	json.Unmarshal(createW.Body.Bytes(), &created)
	performRequest(r, "POST", "/api/v1/purchase/orders/"+created.ID+"/confirm", nil)

	w := performRequest(r, "POST", "/api/v1/purchase/orders/"+created.ID+"/confirm", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

// --- Full Workflow Test ---

func TestAPI_PurchaseWorkflow_FullCycle(t *testing.T) {
	_, r := setupTestHandler(t)

	// 1. Create supplier
	supW := performRequest(r, "POST", "/api/v1/purchase/suppliers", map[string]any{
		"name": "ABC Company", "tax_code": "0123456789",
	})
	if supW.Code != http.StatusCreated {
		t.Fatalf("create supplier: status = %d, want 201", supW.Code)
	}

	// 2. Create purchase order
	poW := performRequest(r, "POST", "/api/v1/purchase/orders", map[string]any{
		"supplier_code": "NCC-00001",
		"order_date":    "2026-08-30",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	})
	if poW.Code != http.StatusCreated {
		t.Fatalf("create PO: status = %d, want 201", poW.Code)
	}

	// 3. Create receipt
	recW := performRequest(r, "POST", "/api/v1/purchase/receipts", map[string]any{
		"po_id":         "po-123",
		"supplier_code": "NCC-00001",
		"receipt_date":  "2026-09-10",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity_received": 10},
		},
	})
	if recW.Code != http.StatusCreated {
		t.Fatalf("create receipt: status = %d, want 201", recW.Code)
	}

	// 4. Create invoice
	invW := performRequest(r, "POST", "/api/v1/purchase/invoices", map[string]any{
		"supplier_code": "NCC-00001",
		"invoice_date":  "2026-09-10",
		"lines": []map[string]any{
			{"line_no": 1, "item_code": "SP-001", "quantity": 10, "unit_price": 500000},
		},
	})
	if invW.Code != http.StatusCreated {
		t.Fatalf("create invoice: status = %d, want 201", invW.Code)
	}

	// 5. Create payment
	payW := performRequest(r, "POST", "/api/v1/purchase/payments", map[string]any{
		"supplier_code":  "NCC-00001",
		"payment_date":   "2026-10-10",
		"payment_method": "bank_transfer",
		"amount":         map[string]any{"amount_minor": 5000000, "currency": "VND"},
		"applied_invoices": []map[string]any{
			{"invoice_id": "inv-1", "amount_applied": 5000000},
		},
	})
	if payW.Code != http.StatusCreated {
		t.Fatalf("create payment: status = %d, want 201", payW.Code)
	}

	// 6. Verify supplier balance
	balW := performRequest(r, "GET", "/api/v1/purchase/suppliers/balance/NCC-00001", nil)
	if balW.Code != http.StatusOK {
		t.Fatalf("get balance: status = %d, want 200", balW.Code)
	}
}

func TestAPI_404_Author(t *testing.T) {
	_, r := setupTestHandler(t)
	w := performRequest(r, "GET", "/api/v1/purchase/suppliers/nonexistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
