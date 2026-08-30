package sales

import (
	"context"
	"testing"

	"goGL/internal/domain/core"
	"goGL/internal/domain/sales"
)

// --- Invoice tests ---

func TestCreateInvoice_Success(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{}}
	svc := NewService(repo)

	input := &sales.SalesInvoice{
		CustomerCode: "KH-001",
		CustomerName: "ABC Company",
		InvoiceDate:  "2026-08-29",
		Lines: []sales.InvoiceLine{
			{LineNo: 1, ItemCode: "SP-001", ItemName: "Widget A", Quantity: 100, UnitPrice: 500000, Discount: 5, COGSAccount: "6321", RevenueAcct: "5111"},
		},
	}

	result, err := svc.CreateInvoice(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefNo == "" {
		t.Error("expected auto-generated ref no")
	}
	if result.Status != sales.InvoiceDraft {
		t.Errorf("status = %q, want draft", result.Status)
	}
	if result.EInvoiceStatus != sales.EInvoiceNone {
		t.Errorf("e_invoice_status = %q, want none", result.EInvoiceStatus)
	}
}

func TestCreateInvoice_EmptyCustomer(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{}}
	svc := NewService(repo)

	input := &sales.SalesInvoice{
		InvoiceDate: "2026-08-29",
		Lines:       []sales.InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}

	_, err := svc.CreateInvoice(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty customer code")
	}
}

func TestCreateInvoice_EmptyLines(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{}}
	svc := NewService(repo)

	input := &sales.SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
	}

	_, err := svc.CreateInvoice(context.Background(), input, "admin")
	if err != sales.ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestGetInvoice_Success(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{
		"inv-1": {ID: "inv-1", RefNo: "HD-00001", CustomerCode: "KH-001", Status: sales.InvoiceDraft},
	}}
	svc := NewService(repo)

	result, err := svc.GetInvoice(context.Background(), "inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefNo != "HD-00001" {
		t.Errorf("ref_no = %q, want HD-00001", result.RefNo)
	}
}

func TestGetInvoice_NotFound(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{}}
	svc := NewService(repo)

	_, err := svc.GetInvoice(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent invoice")
	}
}

func TestListInvoices_Success(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{
		"inv-1": {ID: "inv-1", CustomerCode: "KH-001", Status: sales.InvoiceDraft},
		"inv-2": {ID: "inv-2", CustomerCode: "KH-002", Status: sales.InvoiceIssued},
	}}
	svc := NewService(repo)

	result, err := svc.ListInvoices(context.Background(), "", "", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("count = %d, want 2", len(result))
	}
}

func TestListInvoices_FilterByCustomer(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{
		"inv-1": {ID: "inv-1", CustomerCode: "KH-001"},
		"inv-2": {ID: "inv-2", CustomerCode: "KH-002"},
	}}
	svc := NewService(repo)

	result, err := svc.ListInvoices(context.Background(), "KH-001", "", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("count = %d, want 1", len(result))
	}
}

func TestListInvoices_FilterByStatus(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{
		"inv-1": {ID: "inv-1", Status: sales.InvoiceDraft},
		"inv-2": {ID: "inv-2", Status: sales.InvoiceIssued},
	}}
	svc := NewService(repo)

	result, err := svc.ListInvoices(context.Background(), "", sales.InvoiceIssued, 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("count = %d, want 1", len(result))
	}
}

func TestDeleteInvoice_Draft(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{
		"inv-1": {ID: "inv-1", Status: sales.InvoiceDraft},
	}}
	svc := NewService(repo)

	err := svc.DeleteInvoice(context.Background(), "inv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteInvoice_Issued(t *testing.T) {
	repo := &mockRepo{invoices: map[string]*sales.SalesInvoice{
		"inv-1": {ID: "inv-1", Status: sales.InvoiceIssued},
	}}
	svc := NewService(repo)

	err := svc.DeleteInvoice(context.Background(), "inv-1")
	if err == nil {
		t.Error("expected error for deleting issued invoice")
	}
}

// --- Order tests ---

func TestCreateOrder_Success(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{}}
	svc := NewService(repo)

	input := &sales.SalesOrder{
		CustomerCode: "KH-001",
		CustomerName: "ABC Company",
		OrderDate:    "2026-08-29",
		DeliveryDate: "2026-09-05",
		Lines: []sales.OrderLine{
			{LineNo: 1, ItemCode: "SP-001", ItemName: "Widget A", Quantity: 10, UnitPrice: 500000},
		},
	}

	result, err := svc.CreateOrder(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefNo == "" {
		t.Error("expected auto-generated ref no")
	}
	if result.Status != sales.OrderDraft {
		t.Errorf("status = %q, want draft", result.Status)
	}
	if result.DeliveryStatus != sales.DeliveryPending {
		t.Errorf("delivery_status = %q, want pending", result.DeliveryStatus)
	}
}

func TestCreateOrder_EmptyCustomer(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{}}
	svc := NewService(repo)

	input := &sales.SalesOrder{
		OrderDate: "2026-08-29",
		Lines:     []sales.OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}

	_, err := svc.CreateOrder(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty customer code")
	}
}

func TestGetOrder_Success(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", RefNo: "DH-00001", CustomerCode: "KH-001"},
	}}
	svc := NewService(repo)

	result, err := svc.GetOrder(context.Background(), "ord-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefNo != "DH-00001" {
		t.Errorf("ref_no = %q, want DH-00001", result.RefNo)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{}}
	svc := NewService(repo)

	_, err := svc.GetOrder(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestListOrders_Success(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", CustomerCode: "KH-001"},
		"ord-2": {ID: "ord-2", CustomerCode: "KH-002"},
	}}
	svc := NewService(repo)

	result, err := svc.ListOrders(context.Background(), "", "", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("count = %d, want 2", len(result))
	}
}

func TestConfirmOrder_Success(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", Status: sales.OrderDraft},
	}}
	svc := NewService(repo)

	result, err := svc.ConfirmOrder(context.Background(), "ord-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != sales.OrderConfirmed {
		t.Errorf("status = %q, want confirmed", result.Status)
	}
}

func TestConfirmOrder_AlreadyConfirmed(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", Status: sales.OrderConfirmed},
	}}
	svc := NewService(repo)

	_, err := svc.ConfirmOrder(context.Background(), "ord-1", "admin")
	if err == nil {
		t.Error("expected error for already confirmed order")
	}
}

func TestCancelOrder_Success(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", Status: sales.OrderDraft},
	}}
	svc := NewService(repo)

	result, err := svc.CancelOrder(context.Background(), "ord-1", "Customer changed mind", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != sales.OrderCancelled {
		t.Errorf("status = %q, want cancelled", result.Status)
	}
}

func TestCancelOrder_Delivered(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", Status: sales.OrderDelivered},
	}}
	svc := NewService(repo)

	_, err := svc.CancelOrder(context.Background(), "ord-1", "reason", "admin")
	if err == nil {
		t.Error("expected error for cancelling delivered order")
	}
}

func TestDeleteOrder_Draft(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", Status: sales.OrderDraft},
	}}
	svc := NewService(repo)

	err := svc.DeleteOrder(context.Background(), "ord-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteOrder_Confirmed(t *testing.T) {
	repo := &mockRepo{orders: map[string]*sales.SalesOrder{
		"ord-1": {ID: "ord-1", Status: sales.OrderConfirmed},
	}}
	svc := NewService(repo)

	err := svc.DeleteOrder(context.Background(), "ord-1")
	if err == nil {
		t.Error("expected error for deleting confirmed order")
	}
}

// --- Return tests ---

func TestCreateReturn_Success(t *testing.T) {
	repo := &mockRepo{
		invoices: map[string]*sales.SalesInvoice{
			"inv-1": {ID: "inv-1", CustomerCode: "KH-001", Status: sales.InvoiceIssued},
		},
		returns: map[string]*sales.SalesReturn{},
	}
	svc := NewService(repo)

	input := &sales.SalesReturn{
		InvoiceID:    "inv-1",
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Reason:       sales.ReturnDefective,
		Lines:        []sales.ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 2, UnitPrice: 500000}},
	}

	result, err := svc.CreateReturn(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefNo == "" {
		t.Error("expected auto-generated ref no")
	}
	if result.Status != sales.ReturnDraft {
		t.Errorf("status = %q, want draft", result.Status)
	}
}

func TestCreateReturn_InvalidInvoice(t *testing.T) {
	repo := &mockRepo{
		invoices: map[string]*sales.SalesInvoice{},
		returns:  map[string]*sales.SalesReturn{},
	}
	svc := NewService(repo)

	input := &sales.SalesReturn{
		InvoiceID:    "nonexistent",
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Reason:       sales.ReturnDefective,
		Lines:        []sales.ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}

	_, err := svc.CreateReturn(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for nonexistent invoice")
	}
}

func TestGetReturn_Success(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{
		"ret-1": {ID: "ret-1", RefNo: "PH-00001", CustomerCode: "KH-001"},
	}}
	svc := NewService(repo)

	result, err := svc.GetReturn(context.Background(), "ret-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RefNo != "PH-00001" {
		t.Errorf("ref_no = %q, want PH-00001", result.RefNo)
	}
}

func TestGetReturn_NotFound(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{}}
	svc := NewService(repo)

	_, err := svc.GetReturn(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent return")
	}
}

func TestListReturns_Success(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{
		"ret-1": {ID: "ret-1", CustomerCode: "KH-001"},
		"ret-2": {ID: "ret-2", CustomerCode: "KH-002"},
	}}
	svc := NewService(repo)

	result, err := svc.ListReturns(context.Background(), "", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("count = %d, want 2", len(result))
	}
}

func TestApproveReturn_Success(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{
		"ret-1": {ID: "ret-1", Status: sales.ReturnDraft},
	}}
	svc := NewService(repo)

	result, err := svc.ApproveReturn(context.Background(), "ret-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != sales.ReturnApproved {
		t.Errorf("status = %q, want approved", result.Status)
	}
}

func TestApproveReturn_AlreadyApproved(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{
		"ret-1": {ID: "ret-1", Status: sales.ReturnApproved},
	}}
	svc := NewService(repo)

	_, err := svc.ApproveReturn(context.Background(), "ret-1", "admin")
	if err == nil {
		t.Error("expected error for already approved return")
	}
}

func TestReceiveReturn_Success(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{
		"ret-1": {ID: "ret-1", Status: sales.ReturnApproved},
	}}
	svc := NewService(repo)

	result, err := svc.ReceiveReturn(context.Background(), "ret-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != sales.ReturnReceived {
		t.Errorf("status = %q, want received", result.Status)
	}
}

func TestReceiveReturn_NotApproved(t *testing.T) {
	repo := &mockRepo{returns: map[string]*sales.SalesReturn{
		"ret-1": {ID: "ret-1", Status: sales.ReturnDraft},
	}}
	svc := NewService(repo)

	_, err := svc.ReceiveReturn(context.Background(), "ret-1", "admin")
	if err == nil {
		t.Error("expected error for receiving non-approved return")
	}
}

func TestGetCustomerBalance_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	result, err := svc.GetCustomerBalance(context.Background(), "KH-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Currency != "VND" {
		t.Errorf("currency = %q, want VND", result.Currency)
	}
}

// --- mockRepo ---

type mockRepo struct {
	invoices map[string]*sales.SalesInvoice
	orders   map[string]*sales.SalesOrder
	returns  map[string]*sales.SalesReturn
	invSeq   int64
	ordSeq   int64
	retSeq   int64
}

func (m *mockRepo) CreateInvoice(_ context.Context, inv *sales.SalesInvoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockRepo) FindInvoiceByID(_ context.Context, id string) (*sales.SalesInvoice, error) {
	if inv, ok := m.invoices[id]; ok {
		return inv, nil
	}
	return nil, sales.ErrNotFound
}

func (m *mockRepo) UpdateInvoice(_ context.Context, inv *sales.SalesInvoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockRepo) DeleteInvoice(_ context.Context, id string) error {
	delete(m.invoices, id)
	return nil
}

func (m *mockRepo) ListInvoices(_ context.Context, customerCode string, status sales.InvoiceStatus, limit, offset int) ([]*sales.SalesInvoice, error) {
	var out []*sales.SalesInvoice
	for _, inv := range m.invoices {
		if customerCode != "" && inv.CustomerCode != customerCode {
			continue
		}
		if status != "" && inv.Status != status {
			continue
		}
		out = append(out, inv)
	}
	return out, nil
}

func (m *mockRepo) NextInvoiceNo(_ context.Context) (int64, error) {
	m.invSeq++
	return m.invSeq, nil
}

func (m *mockRepo) CreateOrder(_ context.Context, o *sales.SalesOrder) error {
	m.orders[o.ID] = o
	return nil
}

func (m *mockRepo) FindOrderByID(_ context.Context, id string) (*sales.SalesOrder, error) {
	if o, ok := m.orders[id]; ok {
		return o, nil
	}
	return nil, sales.ErrNotFound
}

func (m *mockRepo) UpdateOrder(_ context.Context, o *sales.SalesOrder) error {
	m.orders[o.ID] = o
	return nil
}

func (m *mockRepo) DeleteOrder(_ context.Context, id string) error {
	delete(m.orders, id)
	return nil
}

func (m *mockRepo) ListOrders(_ context.Context, customerCode string, status sales.OrderStatus, limit, offset int) ([]*sales.SalesOrder, error) {
	var out []*sales.SalesOrder
	for _, o := range m.orders {
		if customerCode != "" && o.CustomerCode != customerCode {
			continue
		}
		if status != "" && o.Status != status {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func (m *mockRepo) NextOrderNo(_ context.Context) (int64, error) {
	m.ordSeq++
	return m.ordSeq, nil
}

func (m *mockRepo) CreateReturn(_ context.Context, r *sales.SalesReturn) error {
	m.returns[r.ID] = r
	return nil
}

func (m *mockRepo) FindReturnByID(_ context.Context, id string) (*sales.SalesReturn, error) {
	if r, ok := m.returns[id]; ok {
		return r, nil
	}
	return nil, sales.ErrNotFound
}

func (m *mockRepo) UpdateReturn(_ context.Context, r *sales.SalesReturn) error {
	m.returns[r.ID] = r
	return nil
}

func (m *mockRepo) ListReturns(_ context.Context, customerCode string, limit, offset int) ([]*sales.SalesReturn, error) {
	var out []*sales.SalesReturn
	for _, r := range m.returns {
		if customerCode != "" && r.CustomerCode != customerCode {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *mockRepo) NextReturnNo(_ context.Context) (int64, error) {
	m.retSeq++
	return m.retSeq, nil
}

func (m *mockRepo) GetCustomerBalance(_ context.Context, customerCode string) (core.Money, error) {
	return core.Money{AmountMinor: 0, Currency: "VND"}, nil
}
