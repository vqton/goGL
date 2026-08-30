package sales

import (
	"testing"
)

func TestValidateSalesInvoice_Success(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Lines: []InvoiceLine{
			{LineNo: 1, ItemCode: "SP-001", Quantity: 100, UnitPrice: 500000},
		},
	}
	if err := ValidateSalesInvoice(inv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != InvoiceDraft {
		t.Errorf("status = %q, want draft", inv.Status)
	}
}

func TestValidateSalesInvoice_EmptyCustomer(t *testing.T) {
	inv := &SalesInvoice{
		InvoiceDate: "2026-08-29",
		Lines:       []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for empty customer code")
	}
}

func TestValidateSalesInvoice_EmptyLines(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Lines:        []InvoiceLine{},
	}
	if err := ValidateSalesInvoice(inv); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidateSalesInvoice_NilLines(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
	}
	if err := ValidateSalesInvoice(inv); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidateSalesInvoice_EmptyDate(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for empty invoice date")
	}
}

func TestValidateSalesInvoice_InvalidStatus(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Status:       "bogus",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestValidateSalesInvoice_EmptyItemCode(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Lines:        []InvoiceLine{{LineNo: 1, Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for empty item code")
	}
}

func TestValidateSalesInvoice_ZeroQuantity(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 0, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for zero quantity")
	}
}

func TestValidateSalesInvoice_NegativeQuantity(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: -5, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for negative quantity")
	}
}

func TestValidateSalesInvoice_NegativePrice(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode: "KH-001",
		InvoiceDate:  "2026-08-29",
		Lines:        []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: -1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for negative unit price")
	}
}

func TestValidateSalesInvoice_InvalidEInvoiceStatus(t *testing.T) {
	inv := &SalesInvoice{
		CustomerCode:   "KH-001",
		InvoiceDate:    "2026-08-29",
		EInvoiceStatus: "bogus",
		Lines:          []InvoiceLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesInvoice(inv); err == nil {
		t.Error("expected error for invalid e-invoice status")
	}
}

func TestInvoiceStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    InvoiceStatus
		want bool
	}{
		{InvoiceDraft, true},
		{InvoicePending, true},
		{InvoiceIssued, true},
		{InvoicePartial, true},
		{InvoicePaid, true},
		{InvoiceOverdue, true},
		{InvoiceVoided, true},
		{InvoiceReturned, true},
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

func TestValidateSalesOrder_Success(t *testing.T) {
	o := &SalesOrder{
		CustomerCode: "KH-001",
		OrderDate:    "2026-08-29",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 10, UnitPrice: 500000}},
	}
	if err := ValidateSalesOrder(o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.Status != OrderDraft {
		t.Errorf("status = %q, want draft", o.Status)
	}
	if o.DeliveryStatus != DeliveryPending {
		t.Errorf("delivery_status = %q, want pending", o.DeliveryStatus)
	}
}

func TestValidateSalesOrder_EmptyCustomer(t *testing.T) {
	o := &SalesOrder{
		OrderDate: "2026-08-29",
		Lines:     []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesOrder(o); err == nil {
		t.Error("expected error for empty customer code")
	}
}

func TestValidateSalesOrder_EmptyLines(t *testing.T) {
	o := &SalesOrder{
		CustomerCode: "KH-001",
		OrderDate:    "2026-08-29",
		Lines:        []OrderLine{},
	}
	if err := ValidateSalesOrder(o); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidateSalesOrder_EmptyDate(t *testing.T) {
	o := &SalesOrder{
		CustomerCode: "KH-001",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesOrder(o); err == nil {
		t.Error("expected error for empty order date")
	}
}

func TestValidateSalesOrder_InvalidStatus(t *testing.T) {
	o := &SalesOrder{
		CustomerCode: "KH-001",
		OrderDate:    "2026-08-29",
		Status:       "bogus",
		Lines:        []OrderLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesOrder(o); err == nil {
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
		{OrderPartialDel, true},
		{OrderDelivered, true},
		{OrderPartialInv, true},
		{OrderInvoiced, true},
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

func TestDeliveryStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    DeliveryStatus
		want bool
	}{
		{DeliveryPending, true},
		{DeliveryPartial, true},
		{DeliveryInTransit, true},
		{DeliveryDelivered, true},
		{DeliveryCompleted, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("DeliveryStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestValidateSalesReturn_Success(t *testing.T) {
	r := &SalesReturn{
		InvoiceID:    "inv-1",
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Reason:       ReturnDefective,
		Lines:        []ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 2, UnitPrice: 500000}},
	}
	if err := ValidateSalesReturn(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != ReturnDraft {
		t.Errorf("status = %q, want draft", r.Status)
	}
}

func TestValidateSalesReturn_EmptyInvoiceID(t *testing.T) {
	r := &SalesReturn{
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Reason:       ReturnDefective,
		Lines:        []ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesReturn(r); err == nil {
		t.Error("expected error for empty invoice ID")
	}
}

func TestValidateSalesReturn_EmptyCustomer(t *testing.T) {
	r := &SalesReturn{
		InvoiceID:  "inv-1",
		ReturnDate: "2026-08-29",
		Reason:     ReturnDefective,
		Lines:      []ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesReturn(r); err == nil {
		t.Error("expected error for empty customer code")
	}
}

func TestValidateSalesReturn_EmptyLines(t *testing.T) {
	r := &SalesReturn{
		InvoiceID:    "inv-1",
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Reason:       ReturnDefective,
		Lines:        []ReturnLine{},
	}
	if err := ValidateSalesReturn(r); err != ErrEmptyLines {
		t.Errorf("expected ErrEmptyLines, got %v", err)
	}
}

func TestValidateSalesReturn_EmptyDate(t *testing.T) {
	r := &SalesReturn{
		InvoiceID:    "inv-1",
		CustomerCode: "KH-001",
		Reason:       ReturnDefective,
		Lines:        []ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesReturn(r); err == nil {
		t.Error("expected error for empty return date")
	}
}

func TestValidateSalesReturn_EmptyReason(t *testing.T) {
	r := &SalesReturn{
		InvoiceID:    "inv-1",
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Lines:        []ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesReturn(r); err == nil {
		t.Error("expected error for empty return reason")
	}
}

func TestValidateSalesReturn_InvalidStatus(t *testing.T) {
	r := &SalesReturn{
		InvoiceID:    "inv-1",
		CustomerCode: "KH-001",
		ReturnDate:   "2026-08-29",
		Reason:       ReturnDefective,
		Status:       "bogus",
		Lines:        []ReturnLine{{LineNo: 1, ItemCode: "SP-001", Quantity: 1, UnitPrice: 1000}},
	}
	if err := ValidateSalesReturn(r); err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestReturnStatus_IsValid(t *testing.T) {
	tests := []struct {
		s    ReturnStatus
		want bool
	}{
		{ReturnDraft, true},
		{ReturnApproved, true},
		{ReturnReceived, true},
		{ReturnIssued, true},
		{ReturnCompleted, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("ReturnStatus(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}
