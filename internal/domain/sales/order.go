package sales

import (
	"goGL/internal/domain/core"
)

// OrderStatus represents the lifecycle status of a sales order.
type OrderStatus string

const (
	OrderDraft      OrderStatus = "draft"
	OrderConfirmed  OrderStatus = "confirmed"
	OrderPartialDel OrderStatus = "partial_delivery"
	OrderDelivered  OrderStatus = "delivered"
	OrderPartialInv OrderStatus = "partial_invoice"
	OrderInvoiced   OrderStatus = "invoiced"
	OrderCompleted  OrderStatus = "completed"
	OrderCancelled  OrderStatus = "cancelled"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderDraft, OrderConfirmed, OrderPartialDel, OrderDelivered,
		OrderPartialInv, OrderInvoiced, OrderCompleted, OrderCancelled:
		return true
	default:
		return false
	}
}

// DeliveryStatus tracks delivery progress.
type DeliveryStatus string

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryPartial   DeliveryStatus = "partial"
	DeliveryInTransit DeliveryStatus = "in_transit"
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryCompleted DeliveryStatus = "completed"
)

func (s DeliveryStatus) IsValid() bool {
	switch s {
	case DeliveryPending, DeliveryPartial, DeliveryInTransit, DeliveryDelivered, DeliveryCompleted:
		return true
	default:
		return false
	}
}

// OrderLine represents a single line item on a sales order.
type OrderLine struct {
	LineNo       int     `json:"line_no"`
	ItemCode     string  `json:"item_code"`
	ItemName     string  `json:"item_name"`
	Unit         string  `json:"unit"`
	Quantity     int64   `json:"quantity"`
	DeliveredQty int64   `json:"delivered_qty"`
	InvoicedQty  int64   `json:"invoiced_qty"`
	UnitPrice    int64   `json:"unit_price"`
	Discount     float64 `json:"discount"`
	Amount       int64   `json:"amount"`
	VATAmount    int64   `json:"vat_amount"`
	TotalAmount  int64   `json:"total_amount"`
}

// SalesOrder represents a sales order (Đơn hàng bán).
type SalesOrder struct {
	ID              string         `json:"id"`
	RefNo           string         `json:"ref_no"` // Auto: DH-XXXXX
	QuoteID         string         `json:"quote_id,omitempty"`
	CustomerCode    string         `json:"customer_code"`
	CustomerName    string         `json:"customer_name"`
	OrderDate       string         `json:"order_date"`
	DeliveryDate    string         `json:"delivery_date"`
	Status          OrderStatus    `json:"status"`
	Lines           []OrderLine    `json:"lines"`
	SubTotal        core.Money     `json:"sub_total"`
	DiscountRate    float64        `json:"discount_rate"`
	DiscountAmount  core.Money     `json:"discount_amount"`
	VATAmount       core.Money     `json:"vat_amount"`
	TotalAmount     core.Money     `json:"total_amount"`
	DeliveryAddress string         `json:"delivery_address"`
	PaymentTerms    string         `json:"payment_terms"`
	Notes           string         `json:"notes,omitempty"`
	DeliveryStatus  DeliveryStatus `json:"delivery_status"`
	DeliveredAmount core.Money     `json:"delivered_amount"`
	InvoicedAmount  core.Money     `json:"invoiced_amount"`
	CreatedBy       string         `json:"created_by,omitempty"`
	CreatedAt       string         `json:"created_at"`
	UpdatedBy       string         `json:"updated_by,omitempty"`
	UpdatedAt       string         `json:"updated_at"`
}

func (o *SalesOrder) Clone() *SalesOrder {
	cp := *o
	if o.Lines != nil {
		cp.Lines = make([]OrderLine, len(o.Lines))
		copy(cp.Lines, o.Lines)
	}
	return &cp
}

// ValidateSalesOrder validates order data.
func ValidateSalesOrder(o *SalesOrder) error {
	if o.CustomerCode == "" {
		return &core.ValidationError{Field: "customer_code", Message: "customer code is required"}
	}
	if len(o.Lines) == 0 {
		return ErrEmptyLines
	}
	if o.OrderDate == "" {
		return &core.ValidationError{Field: "order_date", Message: "order date is required"}
	}
	if o.Status == "" {
		o.Status = OrderDraft
	}
	if !o.Status.IsValid() {
		return &core.ValidationError{Field: "status", Message: "invalid status"}
	}
	if o.DeliveryStatus == "" {
		o.DeliveryStatus = DeliveryPending
	}
	if !o.DeliveryStatus.IsValid() {
		return &core.ValidationError{Field: "delivery_status", Message: "invalid delivery status"}
	}
	for _, line := range o.Lines {
		if line.ItemCode == "" {
			return &core.ValidationError{Field: "item_code", Message: "item code is required"}
		}
		if line.Quantity <= 0 {
			return &core.ValidationError{Field: "quantity", Message: "quantity must be positive"}
		}
	}
	return nil
}
