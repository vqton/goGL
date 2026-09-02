package purchase

import (
	"context"
	"testing"

	"goGL/internal/domain/core"
	"goGL/internal/domain/purchase"
)

// --- mockRepository implements purchase.Repository for testing ---

type mockRepository struct {
	suppliers   map[string]*purchase.Supplier
	orders      map[string]*purchase.PurchaseOrder
	receipts    map[string]*purchase.GoodsReceipt
	invoices    map[string]*purchase.PurchaseInvoice
	payments    map[string]*purchase.Payment
	seqSupplier int64
	seqOrder    int64
	seqReceipt  int64
	seqInvoice  int64
	seqPayment  int64
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		suppliers: make(map[string]*purchase.Supplier),
		orders:    make(map[string]*purchase.PurchaseOrder),
		receipts:  make(map[string]*purchase.GoodsReceipt),
		invoices:  make(map[string]*purchase.PurchaseInvoice),
		payments:  make(map[string]*purchase.Payment),
	}
}

func (m *mockRepository) NextSupplierNo(ctx context.Context) (int64, error) {
	m.seqSupplier++
	return m.seqSupplier, nil
}
func (m *mockRepository) NextOrderNo(ctx context.Context) (int64, error) {
	m.seqOrder++
	return m.seqOrder, nil
}
func (m *mockRepository) NextReceiptNo(ctx context.Context) (int64, error) {
	m.seqReceipt++
	return m.seqReceipt, nil
}
func (m *mockRepository) NextInvoiceNo(ctx context.Context) (int64, error) {
	m.seqInvoice++
	return m.seqInvoice, nil
}
func (m *mockRepository) NextPaymentNo(ctx context.Context) (int64, error) {
	m.seqPayment++
	return m.seqPayment, nil
}

func (m *mockRepository) CreateSupplier(ctx context.Context, s *purchase.Supplier) error {
	m.suppliers[s.ID] = s
	return nil
}
func (m *mockRepository) FindSupplierByID(ctx context.Context, id string) (*purchase.Supplier, error) {
	s, ok := m.suppliers[id]
	if !ok {
		return nil, purchase.ErrNotFound
	}
	return s, nil
}
func (m *mockRepository) UpdateSupplier(ctx context.Context, s *purchase.Supplier) error {
	m.suppliers[s.ID] = s
	return nil
}
func (m *mockRepository) DeleteSupplier(ctx context.Context, id string) error {
	if _, ok := m.suppliers[id]; !ok {
		return purchase.ErrNotFound
	}
	delete(m.suppliers, id)
	return nil
}
func (m *mockRepository) ListSuppliers(ctx context.Context, name string, status purchase.SupplierStatus, limit, offset int) ([]*purchase.Supplier, error) {
	var out []*purchase.Supplier
	for _, s := range m.suppliers {
		if name != "" && s.Name != name {
			continue
		}
		if status != "" && s.Status != status {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (m *mockRepository) CreateOrder(ctx context.Context, o *purchase.PurchaseOrder) error {
	m.orders[o.ID] = o
	return nil
}
func (m *mockRepository) FindOrderByID(ctx context.Context, id string) (*purchase.PurchaseOrder, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, purchase.ErrNotFound
	}
	return o, nil
}
func (m *mockRepository) UpdateOrder(ctx context.Context, o *purchase.PurchaseOrder) error {
	m.orders[o.ID] = o
	return nil
}
func (m *mockRepository) DeleteOrder(ctx context.Context, id string) error {
	if _, ok := m.orders[id]; !ok {
		return purchase.ErrNotFound
	}
	delete(m.orders, id)
	return nil
}
func (m *mockRepository) ListOrders(ctx context.Context, supplierCode string, status purchase.OrderStatus, limit, offset int) ([]*purchase.PurchaseOrder, error) {
	var out []*purchase.PurchaseOrder
	for _, o := range m.orders {
		if supplierCode != "" && o.SupplierCode != supplierCode {
			continue
		}
		if status != "" && o.Status != status {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func (m *mockRepository) CreateReceipt(ctx context.Context, g *purchase.GoodsReceipt) error {
	m.receipts[g.ID] = g
	return nil
}
func (m *mockRepository) FindReceiptByID(ctx context.Context, id string) (*purchase.GoodsReceipt, error) {
	g, ok := m.receipts[id]
	if !ok {
		return nil, purchase.ErrNotFound
	}
	return g, nil
}
func (m *mockRepository) UpdateReceipt(ctx context.Context, g *purchase.GoodsReceipt) error {
	m.receipts[g.ID] = g
	return nil
}
func (m *mockRepository) ListReceipts(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.GoodsReceipt, error) {
	var out []*purchase.GoodsReceipt
	for _, g := range m.receipts {
		if supplierCode != "" && g.SupplierCode != supplierCode {
			continue
		}
		out = append(out, g)
	}
	return out, nil
}
func (m *mockRepository) HasReceiptsForOrder(ctx context.Context, poID string) (bool, error) {
	for _, g := range m.receipts {
		if g.POID == poID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) CreateInvoice(ctx context.Context, inv *purchase.PurchaseInvoice) error {
	m.invoices[inv.ID] = inv
	return nil
}
func (m *mockRepository) FindInvoiceByID(ctx context.Context, id string) (*purchase.PurchaseInvoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, purchase.ErrNotFound
	}
	return inv, nil
}
func (m *mockRepository) UpdateInvoice(ctx context.Context, inv *purchase.PurchaseInvoice) error {
	m.invoices[inv.ID] = inv
	return nil
}
func (m *mockRepository) DeleteInvoice(ctx context.Context, id string) error {
	if _, ok := m.invoices[id]; !ok {
		return purchase.ErrNotFound
	}
	delete(m.invoices, id)
	return nil
}
func (m *mockRepository) ListInvoices(ctx context.Context, supplierCode string, status purchase.InvoiceStatus, limit, offset int) ([]*purchase.PurchaseInvoice, error) {
	var out []*purchase.PurchaseInvoice
	for _, inv := range m.invoices {
		if supplierCode != "" && inv.SupplierCode != supplierCode {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		out = append(out, inv)
	}
	return out, nil
}

func (m *mockRepository) CreatePayment(ctx context.Context, p *purchase.Payment) error {
	m.payments[p.ID] = p
	return nil
}
func (m *mockRepository) FindPaymentByID(ctx context.Context, id string) (*purchase.Payment, error) {
	p, ok := m.payments[id]
	if !ok {
		return nil, purchase.ErrNotFound
	}
	return p, nil
}
func (m *mockRepository) UpdatePayment(ctx context.Context, p *purchase.Payment) error {
	m.payments[p.ID] = p
	return nil
}
func (m *mockRepository) ListPayments(ctx context.Context, supplierCode string, limit, offset int) ([]*purchase.Payment, error) {
	var out []*purchase.Payment
	for _, p := range m.payments {
		if supplierCode != "" && p.SupplierCode != supplierCode {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
func (m *mockRepository) GetSupplierBalance(ctx context.Context, supplierCode string) (core.Money, error) {
	return core.Money{AmountMinor: 0, Currency: "VND"}, nil
}

// --- Supplier Service Tests ---

func TestCreateSupplier_Success(t *testing.T) {
	svc := NewService(newMockRepo())
	s := &purchase.Supplier{
		Name:    "ABC Company",
		TaxCode: "0123456789",
	}
	got, err := svc.CreateSupplier(context.Background(), s, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == "" {
		t.Error("expected ID to be set")
	}
	if got.RefNo != "NCC-00001" {
		t.Errorf("RefNo = %q, want NCC-00001", got.RefNo)
	}
	if got.Status != purchase.SupplierActive {
		t.Errorf("Status = %q, want active", got.Status)
	}
	if got.CreatedBy != "admin" {
		t.Errorf("CreatedBy = %q, want admin", got.CreatedBy)
	}
}

func TestCreateSupplier_ValidationError(t *testing.T) {
	svc := NewService(newMockRepo())
	s := &purchase.Supplier{
		TaxCode: "0123456789",
	}
	_, err := svc.CreateSupplier(context.Background(), s, "admin")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestGetSupplier_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	s := &purchase.Supplier{Name: "ABC", TaxCode: "0123456789"}
	created, _ := svc.CreateSupplier(context.Background(), s, "admin")

	got, err := svc.GetSupplier(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "ABC" {
		t.Errorf("Name = %q, want ABC", got.Name)
	}
}

func TestGetSupplier_NotFound(t *testing.T) {
	svc := NewService(newMockRepo())
	_, err := svc.GetSupplier(context.Background(), "nonexistent")
	if err != purchase.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateSupplier_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	s := &purchase.Supplier{Name: "ABC", TaxCode: "0123456789"}
	created, _ := svc.CreateSupplier(context.Background(), s, "admin")

	patch := &purchase.Supplier{
		Name:    "XYZ Company",
		TaxCode: "9876543210",
	}
	got, err := svc.UpdateSupplier(context.Background(), created.ID, patch, "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "XYZ Company" {
		t.Errorf("Name = %q, want XYZ Company", got.Name)
	}
	if got.UpdatedBy != "user1" {
		t.Errorf("UpdatedBy = %q, want user1", got.UpdatedBy)
	}
}

func TestDeleteSupplier_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	s := &purchase.Supplier{Name: "ABC", TaxCode: "0123456789"}
	created, _ := svc.CreateSupplier(context.Background(), s, "admin")

	if err := svc.DeleteSupplier(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListSuppliers_DefaultLimit(t *testing.T) {
	svc := NewService(newMockRepo())
	s := &purchase.Supplier{Name: "ABC", TaxCode: "0123456789"}
	svc.CreateSupplier(context.Background(), s, "admin")

	got, err := svc.ListSuppliers(context.Background(), "", "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d suppliers, want 1", len(got))
	}
}

// --- Purchase Order Service Tests ---

func TestCreateOrder_Success(t *testing.T) {
	svc := NewService(newMockRepo())
	o := &purchase.PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	got, err := svc.CreateOrder(context.Background(), o, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RefNo != "PO-00001" {
		t.Errorf("RefNo = %q, want PO-00001", got.RefNo)
	}
	if got.Status != purchase.OrderDraft {
		t.Errorf("Status = %q, want draft", got.Status)
	}
}

func TestCreateOrder_ValidationError(t *testing.T) {
	svc := NewService(newMockRepo())
	o := &purchase.PurchaseOrder{
		OrderDate: "2026-08-30",
		Lines:     []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	_, err := svc.CreateOrder(context.Background(), o, "admin")
	if err == nil {
		t.Fatal("expected error for empty supplier code")
	}
}

func TestConfirmOrder_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	o := &purchase.PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateOrder(context.Background(), o, "admin")

	got, err := svc.ConfirmOrder(context.Background(), created.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != purchase.OrderConfirmed {
		t.Errorf("Status = %q, want confirmed", got.Status)
	}
}

func TestConfirmOrder_WrongStatus(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	o := &purchase.PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateOrder(context.Background(), o, "admin")
	svc.ConfirmOrder(context.Background(), created.ID, "admin")

	_, err := svc.ConfirmOrder(context.Background(), created.ID, "admin")
	if err != purchase.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestCancelOrder_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	o := &purchase.PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateOrder(context.Background(), o, "admin")

	got, err := svc.CancelOrder(context.Background(), created.ID, "changed mind", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != purchase.OrderCancelled {
		t.Errorf("Status = %q, want cancelled", got.Status)
	}
}

func TestDeleteOrder_DraftOnly(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	o := &purchase.PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateOrder(context.Background(), o, "admin")
	svc.ConfirmOrder(context.Background(), created.ID, "admin")

	err := svc.DeleteOrder(context.Background(), created.ID)
	if err != purchase.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestCancelOrder_WithReceipt_Blocked(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	o := &purchase.PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []purchase.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateOrder(context.Background(), o, "admin")

	// Create a receipt linked to this PO
	repo.receipts["gr-1"] = &purchase.GoodsReceipt{
		ID: "gr-1", POID: created.ID, SupplierCode: "NCC-00001",
	}

	_, err := svc.CancelOrder(context.Background(), created.ID, "changed mind", "admin")
	if err != purchase.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus when receipts exist, got %v", err)
	}
}

// --- Goods Receipt Service Tests ---

func TestCreateReceipt_Success(t *testing.T) {
	svc := NewService(newMockRepo())
	g := &purchase.GoodsReceipt{
		POID:         "po-123",
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Lines:        []purchase.ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	got, err := svc.CreateReceipt(context.Background(), g, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RefNo != "NK-00001" {
		t.Errorf("RefNo = %q, want NK-00001", got.RefNo)
	}
	if got.Status != purchase.ReceiptDraft {
		t.Errorf("Status = %q, want draft", got.Status)
	}
}

func TestApproveReceipt_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	g := &purchase.GoodsReceipt{
		POID:         "po-123",
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Lines:        []purchase.ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	created, _ := svc.CreateReceipt(context.Background(), g, "admin")

	got, err := svc.ApproveReceipt(context.Background(), created.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != purchase.ReceiptApproved {
		t.Errorf("Status = %q, want approved", got.Status)
	}
}

// --- Purchase Invoice Service Tests ---

func TestCreateInvoice_Success(t *testing.T) {
	svc := NewService(newMockRepo())
	inv := &purchase.PurchaseInvoice{
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Lines:        []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	got, err := svc.CreateInvoice(context.Background(), inv, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RefNo != "MH-00001" {
		t.Errorf("RefNo = %q, want MH-00001", got.RefNo)
	}
	if got.Status != purchase.InvoiceDraft {
		t.Errorf("Status = %q, want draft", got.Status)
	}
	if got.EInvoiceStatus != purchase.EInvoiceNone {
		t.Errorf("EInvoiceStatus = %q, want none", got.EInvoiceStatus)
	}
}

func TestPostInvoice_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	inv := &purchase.PurchaseInvoice{
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Lines:        []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateInvoice(context.Background(), inv, "admin")

	got, err := svc.PostInvoice(context.Background(), created.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != purchase.InvoicePendingEInv {
		t.Errorf("Status = %q, want pending_einv", got.Status)
	}
}

func TestDeleteInvoice_DraftOnly(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	inv := &purchase.PurchaseInvoice{
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Lines:        []purchase.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	created, _ := svc.CreateInvoice(context.Background(), inv, "admin")
	svc.PostInvoice(context.Background(), created.ID, "admin")

	err := svc.DeleteInvoice(context.Background(), created.ID)
	if err != purchase.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

// --- Payment Service Tests ---

func TestCreatePayment_Success(t *testing.T) {
	svc := NewService(newMockRepo())
	p := &purchase.Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   purchase.PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []purchase.PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	got, err := svc.CreatePayment(context.Background(), p, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RefNo != "TT-00001" {
		t.Errorf("RefNo = %q, want TT-00001", got.RefNo)
	}
	if got.Status != purchase.PaymentDraft {
		t.Errorf("Status = %q, want draft", got.Status)
	}
}

func TestApprovePayment_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	p := &purchase.Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   purchase.PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []purchase.PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	created, _ := svc.CreatePayment(context.Background(), p, "admin")

	got, err := svc.ApprovePayment(context.Background(), created.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != purchase.PaymentApproved {
		t.Errorf("Status = %q, want approved", got.Status)
	}
}

func TestApprovePayment_WrongStatus(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	p := &purchase.Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   purchase.PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []purchase.PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	created, _ := svc.CreatePayment(context.Background(), p, "admin")
	svc.ApprovePayment(context.Background(), created.ID, "admin")

	_, err := svc.ApprovePayment(context.Background(), created.ID, "admin")
	if err != purchase.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

// --- Supplier Balance Test ---

func TestGetSupplierBalance_Success(t *testing.T) {
	svc := NewService(newMockRepo())
	balance, err := svc.GetSupplierBalance(context.Background(), "NCC-00001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if balance.AmountMinor != 0 {
		t.Errorf("AmountMinor = %d, want 0", balance.AmountMinor)
	}
}
