package purchase

import (
	"testing"

	"goGL/internal/domain/core"
)

// --- Supplier tests ---

func TestValidateSupplier_Success(t *testing.T) {
	s := &Supplier{
		Name:    "ABC Company",
		TaxCode: "0123456789",
	}
	if err := ValidateSupplier(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status != SupplierActive {
		t.Errorf("status = %q, want active", s.Status)
	}
}

func TestValidateSupplier_EmptyName(t *testing.T) {
	s := &Supplier{
		TaxCode: "0123456789",
	}
	if err := ValidateSupplier(s); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidateSupplier_EmptyTaxCode(t *testing.T) {
	s := &Supplier{
		Name: "ABC Company",
	}
	if err := ValidateSupplier(s); err == nil {
		t.Error("expected error for empty tax code")
	}
}

func TestValidateSupplier_InvalidStatus(t *testing.T) {
	s := &Supplier{
		Name:    "ABC Company",
		TaxCode: "0123456789",
		Status:  "bogus",
	}
	if err := ValidateSupplier(s); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestSupplierStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    SupplierStatus
		want bool
	}{
		{SupplierActive, true},
		{SupplierInactive, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("SupplierStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestCloneSupplier(t *testing.T) {
	s := &Supplier{
		Name:    "ABC Company",
		TaxCode: "0123456789",
	}
	cp := s.Clone()
	cp.Name = "XYZ Company"
	if s.Name == cp.Name {
		t.Error("clone should not modify original")
	}
}

// --- PurchaseOrder tests ---

func TestValidatePurchaseOrder_Success(t *testing.T) {
	o := &PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	if err := ValidatePurchaseOrder(o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.Status != OrderDraft {
		t.Errorf("status = %q, want draft", o.Status)
	}
}

func TestValidatePurchaseOrder_EmptySupplier(t *testing.T) {
	o := &PurchaseOrder{
		OrderDate: "2026-08-30",
		Lines:     []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseOrder(o); err == nil {
		t.Error("expected error for empty supplier code")
	}
}

func TestValidatePurchaseOrder_EmptyLines(t *testing.T) {
	o := &PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Lines:        []OrderLine{},
	}
	if err := ValidatePurchaseOrder(o); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidatePurchaseOrder_EmptyDate(t *testing.T) {
	o := &PurchaseOrder{
		SupplierCode: "NCC-00001",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseOrder(o); err == nil {
		t.Error("expected error for empty order date")
	}
}

func TestValidatePurchaseOrder_InvalidStatus(t *testing.T) {
	o := &PurchaseOrder{
		SupplierCode: "NCC-00001",
		OrderDate:    "2026-08-30",
		Status:       "bogus",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseOrder(o); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestOrderStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    OrderStatus
		want bool
	}{
		{OrderDraft, true},
		{OrderConfirmed, true},
		{OrderPartial, true},
		{OrderReceived, true},
		{OrderCompleted, true},
		{OrderCancelled, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("OrderStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestClonePurchaseOrder(t *testing.T) {
	o := &PurchaseOrder{
		SupplierCode: "NCC-00001",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001"}},
	}
	cp := o.Clone()
	cp.Lines[0].ItemCode = "SP-002"
	if o.Lines[0].ItemCode == cp.Lines[0].ItemCode {
		t.Error("clone should not modify original")
	}
}

// --- GoodsReceipt tests ---

func TestValidateGoodsReceipt_Success(t *testing.T) {
	r := &GoodsReceipt{
		POID:         "po-123",
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Lines:        []ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	if err := ValidateGoodsReceipt(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != ReceiptDraft {
		t.Errorf("status = %q, want draft", r.Status)
	}
}

func TestValidateGoodsReceipt_EmptyPOID(t *testing.T) {
	r := &GoodsReceipt{
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Lines:        []ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	if err := ValidateGoodsReceipt(r); err == nil {
		t.Error("expected error for empty PO ID")
	}
}

func TestValidateGoodsReceipt_EmptySupplier(t *testing.T) {
	r := &GoodsReceipt{
		POID:        "po-123",
		ReceiptDate: "2026-09-10",
		Lines:       []ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	if err := ValidateGoodsReceipt(r); err == nil {
		t.Error("expected error for empty supplier code")
	}
}

func TestValidateGoodsReceipt_EmptyLines(t *testing.T) {
	r := &GoodsReceipt{
		POID:         "po-123",
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Lines:        []ReceiptLine{},
	}
	if err := ValidateGoodsReceipt(r); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidateGoodsReceipt_EmptyDate(t *testing.T) {
	r := &GoodsReceipt{
		POID:         "po-123",
		SupplierCode: "NCC-00001",
		Lines:        []ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	if err := ValidateGoodsReceipt(r); err == nil {
		t.Error("expected error for empty receipt date")
	}
}

func TestValidateGoodsReceipt_InvalidStatus(t *testing.T) {
	r := &GoodsReceipt{
		POID:         "po-123",
		SupplierCode: "NCC-00001",
		ReceiptDate:  "2026-09-10",
		Status:       "bogus",
		Lines:        []ReceiptLine{{LineNo: 1, ItemCode: "SP-001", QuantityReceived: 10}},
	}
	if err := ValidateGoodsReceipt(r); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestReceiptStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    ReceiptStatus
		want bool
	}{
		{ReceiptDraft, true},
		{ReceiptApproved, true},
		{ReceiptCompleted, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("ReceiptStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// --- PurchaseInvoice tests ---

func TestValidatePurchaseInvoice_Success(t *testing.T) {
	inv := &PurchaseInvoice{
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	if err := ValidatePurchaseInvoice(inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != InvoiceDraft {
		t.Errorf("status = %q, want draft", inv.Status)
	}
	if inv.EInvoiceStatus != EInvoiceNone {
		t.Errorf("e_invoice_status = %q, want none", inv.EInvoiceStatus)
	}
}

func TestValidatePurchaseInvoice_EmptySupplier(t *testing.T) {
	inv := &PurchaseInvoice{
		InvoiceDate: "2026-09-10",
		Lines:       []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseInvoice(inv); err == nil {
		t.Error("expected error for empty supplier code")
	}
}

func TestValidatePurchaseInvoice_EmptyLines(t *testing.T) {
	inv := &PurchaseInvoice{
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Lines:        []InvoiceLine{},
	}
	if err := ValidatePurchaseInvoice(inv); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidatePurchaseInvoice_EmptyDate(t *testing.T) {
	inv := &PurchaseInvoice{
		SupplierCode: "NCC-00001",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseInvoice(inv); err == nil {
		t.Error("expected error for empty invoice date")
	}
}

func TestValidatePurchaseInvoice_InvalidStatus(t *testing.T) {
	inv := &PurchaseInvoice{
		SupplierCode: "NCC-00001",
		InvoiceDate:  "2026-09-10",
		Status:       "bogus",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseInvoice(inv); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestValidatePurchaseInvoice_InvalidEInvoiceStatus(t *testing.T) {
	inv := &PurchaseInvoice{
		SupplierCode:   "NCC-00001",
		InvoiceDate:    "2026-09-10",
		EInvoiceStatus: "bogus",
		Lines:          []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidatePurchaseInvoice(inv); err == nil {
		t.Error("expected error for invalid e-invoice status")
	}
}

func TestInvoiceStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    InvoiceStatus
		want bool
	}{
		{InvoiceDraft, true},
		{InvoicePendingEInv, true},
		{InvoiceValidated, true},
		{InvoicePosted, true},
		{InvoicePaid, true},
		{InvoiceReconciled, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("InvoiceStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestEInvoiceStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    EInvoiceStatus
		want bool
	}{
		{EInvoiceNone, true},
		{EInvoicePending, true},
		{EInvoiceAccepted, true},
		{EInvoiceRejected, true},
		{EInvoiceCancelled, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("EInvoiceStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// --- Payment tests ---

func TestValidatePayment_Success(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	if err := ValidatePayment(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != PaymentDraft {
		t.Errorf("status = %q, want draft", p.Status)
	}
}

func TestValidatePayment_EmptySupplier(t *testing.T) {
	p := &Payment{
		PaymentDate:     "2026-10-10",
		PaymentMethod:   PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for empty supplier code")
	}
}

func TestValidatePayment_EmptyDate(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentMethod:   PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for empty payment date")
	}
}

func TestValidatePayment_EmptyMethod(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for empty payment method")
	}
}

func TestValidatePayment_InvalidMethod(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   "bogus",
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for invalid payment method")
	}
}

func TestValidatePayment_ZeroAmount(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 0, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 0}},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestValidatePayment_EmptyInvoices(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		AppliedInvoices: []PaymentApplication{},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for empty applied invoices")
	}
}

func TestValidatePayment_InvalidStatus(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		PaymentDate:     "2026-10-10",
		PaymentMethod:   PaymentBankTransfer,
		Amount:          core.Money{AmountMinor: 5000000, Currency: "VND"},
		Status:          "bogus",
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	if err := ValidatePayment(p); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestPaymentStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    PaymentStatus
		want bool
	}{
		{PaymentDraft, true},
		{PaymentApproved, true},
		{PaymentProcessed, true},
		{PaymentReconciled, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("PaymentStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestPaymentMethod_IsValid(t *testing.T) {
	tests := []struct {
		m    PaymentMethod
		want bool
	}{
		{PaymentBankTransfer, true},
		{PaymentCash, true},
		{PaymentCheque, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.m.IsValid(); got != tt.want {
			t.Errorf("PaymentMethod(%q).IsValid() = %v, want %v", tt.m, got, tt.want)
		}
	}
}

func TestClonePayment(t *testing.T) {
	p := &Payment{
		SupplierCode:    "NCC-00001",
		AppliedInvoices: []PaymentApplication{{InvoiceID: "inv-1", AmountApplied: 5000000}},
	}
	cp := p.Clone()
	cp.AppliedInvoices[0].InvoiceID = "inv-2"
	if p.AppliedInvoices[0].InvoiceID == cp.AppliedInvoices[0].InvoiceID {
		t.Error("clone should not modify original")
	}
}
