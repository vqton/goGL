package purchase

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"goGL/internal/domain/core"
	"goGL/internal/domain/purchase"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE suppliers (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create suppliers table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE supplier_sequences (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create supplier_sequences table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_orders (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_orders table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_order_sequences (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_order_sequences table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE goods_receipts (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create goods_receipts table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE goods_receipt_sequences (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create goods_receipt_sequences table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_invoices (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_invoices table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_invoice_sequences (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_invoice_sequences table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_payments (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_payments table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_payment_sequences (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_payment_sequences table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE purchase_sequences (id TEXT PRIMARY KEY, data TEXT NOT NULL)`); err != nil {
		t.Fatalf("failed to create purchase_sequences table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- Supplier Repository Tests ---

func TestRepo_CreateSupplier_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	s := &purchase.Supplier{
		ID:      "supplier-1",
		RefNo:   "NCC-00001",
		Name:    "ABC Company",
		TaxCode: "0123456789",
		Status:  purchase.SupplierActive,
	}
	if err := repo.CreateSupplier(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := repo.FindSupplierByID(context.Background(), "supplier-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "ABC Company" {
		t.Errorf("Name = %q, want ABC Company", got.Name)
	}
	if got.TaxCode != "0123456789" {
		t.Errorf("TaxCode = %q, want 0123456789", got.TaxCode)
	}
}

func TestRepo_FindSupplier_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	_, err := repo.FindSupplierByID(context.Background(), "nonexistent")
	if err != purchase.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepo_UpdateSupplier_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	s := &purchase.Supplier{
		ID:      "supplier-1",
		RefNo:   "NCC-00001",
		Name:    "ABC Company",
		TaxCode: "0123456789",
		Status:  purchase.SupplierActive,
	}
	repo.CreateSupplier(context.Background(), s)

	s.Name = "XYZ Company"
	if err := repo.UpdateSupplier(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := repo.FindSupplierByID(context.Background(), "supplier-1")
	if got.Name != "XYZ Company" {
		t.Errorf("Name = %q, want XYZ Company", got.Name)
	}
}

func TestRepo_DeleteSupplier_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	s := &purchase.Supplier{
		ID:      "supplier-1",
		RefNo:   "NCC-00001",
		Name:    "ABC Company",
		TaxCode: "0123456789",
		Status:  purchase.SupplierActive,
	}
	repo.CreateSupplier(context.Background(), s)

	if err := repo.DeleteSupplier(context.Background(), "supplier-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := repo.FindSupplierByID(context.Background(), "supplier-1")
	if err != purchase.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRepo_ListSuppliers_Filter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	repo.CreateSupplier(context.Background(), &purchase.Supplier{
		ID: "s1", RefNo: "NCC-00001", Name: "Alpha", TaxCode: "001", Status: purchase.SupplierActive,
	})
	repo.CreateSupplier(context.Background(), &purchase.Supplier{
		ID: "s2", RefNo: "NCC-00002", Name: "Beta", TaxCode: "002", Status: purchase.SupplierInactive,
	})

	got, err := repo.ListSuppliers(context.Background(), "", purchase.SupplierActive, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d suppliers, want 1", len(got))
	}
	if got[0].Name != "Alpha" {
		t.Errorf("Name = %q, want Alpha", got[0].Name)
	}
}

func TestRepo_NextSupplierNo_Sequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	n1, _ := repo.NextSupplierNo(context.Background())
	n2, _ := repo.NextSupplierNo(context.Background())
	n3, _ := repo.NextSupplierNo(context.Background())
	if n1 != 1 || n2 != 2 || n3 != 3 {
		t.Errorf("seq = %d,%d,%d, want 1,2,3", n1, n2, n3)
	}
}

// --- Purchase Order Repository Tests ---

func TestRepo_CreateOrder_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	o := &purchase.PurchaseOrder{
		ID:           "po-1",
		RefNo:        "PO-00001",
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Status:       purchase.OrderDraft,
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	if err := repo.CreateOrder(context.Background(), o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := repo.FindOrderByID(context.Background(), "po-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SupplierCode != "NCC-00001" {
		t.Errorf("SupplierCode = %q, want NCC-00001", got.SupplierCode)
	}
	if len(got.Lines) != 1 {
		t.Errorf("Lines = %d, want 1", len(got.Lines))
	}
}

func TestRepo_UpdateOrder_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	o := &purchase.PurchaseOrder{
		ID:           "po-1",
		RefNo:        "PO-00001",
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Status:       purchase.OrderDraft,
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	repo.CreateOrder(context.Background(), o)

	o.Status = purchase.OrderConfirmed
	if err := repo.UpdateOrder(context.Background(), o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := repo.FindOrderByID(context.Background(), "po-1")
	if got.Status != purchase.OrderConfirmed {
		t.Errorf("Status = %q, want confirmed", got.Status)
	}
}

func TestRepo_ListOrders_Filter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	repo.CreateOrder(context.Background(), &purchase.PurchaseOrder{
		ID: "po-1", RefNo: "PO-00001", SupplierCode: "NCC-00001", Status: purchase.OrderDraft,
		Lines: []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 100}},
	})
	repo.CreateOrder(context.Background(), &purchase.PurchaseOrder{
		ID: "po-2", RefNo: "PO-00002", SupplierCode: "NCC-00002", Status: purchase.OrderConfirmed,
		Lines: []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-002", Quantity: 1, UnitPrice: 200}},
	})

	got, err := repo.ListOrders(context.Background(), "NCC-00001", "", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d orders, want 1", len(got))
	}
	if got[0].SupplierCode != "NCC-00001" {
		t.Errorf("SupplierCode = %q, want NCC-00001", got[0].SupplierCode)
	}
}

func TestRepo_NextOrderNo_Sequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	n1, _ := repo.NextOrderNo(context.Background())
	n2, _ := repo.NextOrderNo(context.Background())
	if n1 != 1 || n2 != 2 {
		t.Errorf("seq = %d,%d, want 1,2", n1, n2)
	}
}

// --- Goods Receipt Repository Tests ---

func TestRepo_CreateReceipt_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	g := &purchase.GoodsReceipt{
		ID:           "gr-1",
		RefNo:        "NK-00001",
		POID:         "po-1",
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Status:       purchase.ReceiptDraft,
		Lines:        []purchase.ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	if err := repo.CreateReceipt(context.Background(), g); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := repo.FindReceiptByID(context.Background(), "gr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.POID != "po-1" {
		t.Errorf("POID = %q, want po-1", got.POID)
	}
}

func TestRepo_ListReceipts_Filter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	repo.CreateReceipt(context.Background(), &purchase.GoodsReceipt{
		ID: "gr-1", RefNo: "NK-00001", SupplierCode: "NCC-00001", ReceiptDate: "2026-09-10", Status: purchase.ReceiptDraft,
		Lines: []purchase.ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	})
	repo.CreateReceipt(context.Background(), &purchase.GoodsReceipt{
		ID: "gr-2", RefNo: "NK-00002", SupplierCode: "NCC-00002", ReceiptDate: "2026-09-11", Status: purchase.ReceiptApproved,
		Lines: []purchase.ReceiptLine{{LineNo: 1, ItemCode: "SP-002", QuantityReceived: 5}},
	})

	got, err := repo.ListReceipts(context.Background(), "NCC-00001", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d receipts, want 1", len(got))
	}
}

func TestRepo_NextReceiptNo_Sequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	n1, _ := repo.NextReceiptNo(context.Background())
	n2, _ := repo.NextReceiptNo(context.Background())
	if n1 != 1 || n2 != 2 {
		t.Errorf("seq = %d,%d, want 1,2", n1, n2)
	}
}

// --- Purchase Invoice Repository Tests ---

func TestRepo_CreateInvoice_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	inv := &purchase.PurchaseInvoice{
		ID:           "inv-1",
		RefNo:        "MH-00001",
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Status:       purchase.InvoiceDraft,
		Lines:        []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	if err := repo.CreateInvoice(context.Background(), inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := repo.FindInvoiceByID(context.Background(), "inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RefNo != "MH-00001" {
		t.Errorf("RefNo = %q, want MH-00001", got.RefNo)
	}
}

func TestRepo_DeleteInvoice_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	inv := &purchase.PurchaseInvoice{
		ID: "inv-1", RefNo: "MH-00001", SupplierCode: "NCC-00001", InvoiceDate: "2026-09-10", Status: purchase.InvoiceDraft,
		Lines: []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	repo.CreateInvoice(context.Background(), inv)

	if err := repo.DeleteInvoice(context.Background(), "inv-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := repo.FindInvoiceByID(context.Background(), "inv-1")
	if err != purchase.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRepo_ListInvoices_Filter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	repo.CreateInvoice(context.Background(), &purchase.PurchaseInvoice{
		ID: "inv-1", RefNo: "MH-00001", SupplierCode: "NCC-00001", InvoiceDate: "2026-09-10", Status: purchase.InvoiceDraft,
		Lines: []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	})
	repo.CreateInvoice(context.Background(), &purchase.PurchaseInvoice{
		ID: "inv-2", RefNo: "MH-00002", SupplierCode: "NCC-00001", InvoiceDate: "2026-09-11", Status: purchase.InvoicePosted,
		Lines: []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-002", Quantity: 5, UnitPrice: 200000}},
	})

	got, err := repo.ListInvoices(context.Background(), "NCC-00001", purchase.InvoiceDraft, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d invoices, want 1", len(got))
	}
	if got[0].Status != purchase.InvoiceDraft {
		t.Errorf("Status = %q, want draft", got[0].Status)
	}
}

func TestRepo_NextInvoiceNo_Sequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	n1, _ := repo.NextInvoiceNo(context.Background())
	n2, _ := repo.NextInvoiceNo(context.Background())
	if n1 != 1 || n2 != 2 {
		t.Errorf("seq = %d,%d, want 1,2", n1, n2)
	}
}

// --- Payment Repository Tests ---

func TestRepo_CreatePayment_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	p := &purchase.Payment{
		ID:            "pay-1",
		RefNo:         "TT-00001",
		SupplierCode:  "NCC-00001",
		PaymentDate:   "2026-10-10",
		PaymentMethod: purchase.PaymentBankTransfer,
		Amount:        core.Money{AmountMinor: 5000000, Currency: "VND"},
		Status:        purchase.PaymentDraft,
		AppliedInvoices: []purchase.PaymentApplication{
			{InvoiceID: "inv-1", AmountApplied: 5000000},
		},
	}
	if err := repo.CreatePayment(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := repo.FindPaymentByID(context.Background(), "pay-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RefNo != "TT-00001" {
		t.Errorf("RefNo = %q, want TT-00001", got.RefNo)
	}
	if got.Amount.AmountMinor != 5000000 {
		t.Errorf("Amount = %d, want 5000000", got.Amount.AmountMinor)
	}
}

func TestRepo_ListPayments_Filter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	repo.CreatePayment(context.Background(), &purchase.Payment{
		ID: "pay-1", RefNo: "TT-00001", SupplierCode: "NCC-00001", PaymentDate: "2026-10-10",
		PaymentMethod: purchase.PaymentBankTransfer, Amount: core.Money{AmountMinor: 1000000, Currency: "VND"}, Status: purchase.PaymentDraft,
		AppliedInvoices: []purchase.PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 1000000}},
	})
	repo.CreatePayment(context.Background(), &purchase.Payment{
		ID: "pay-2", RefNo: "TT-00002", SupplierCode: "NCC-00002", PaymentDate: "2026-10-11",
		PaymentMethod: purchase.PaymentCash, Amount: core.Money{AmountMinor: 2000000, Currency: "VND"}, Status: purchase.PaymentApproved,
		AppliedInvoices: []purchase.PaymentApplication{{InvoiceID: "inv-2", AmountApplied: 2000000}},
	})

	got, err := repo.ListPayments(context.Background(), "NCC-00001", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d payments, want 1", len(got))
	}
}

func TestRepo_NextPaymentNo_Sequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	n1, _ := repo.NextPaymentNo(context.Background())
	n2, _ := repo.NextPaymentNo(context.Background())
	if n1 != 1 || n2 != 2 {
		t.Errorf("seq = %d,%d, want 1,2", n1, n2)
	}
}

// --- Supplier Balance Test ---

func TestRepo_GetSupplierBalance(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSqliteRepository(db)
	balance, err := repo.GetSupplierBalance(context.Background(), "NCC-00001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if balance.AmountMinor != 0 {
		t.Errorf("AmountMinor = %d, want 0", balance.AmountMinor)
	}
}
