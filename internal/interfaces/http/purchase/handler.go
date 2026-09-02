package purchase

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/purchase"
	"goGL/internal/domain/core"
	domainpurchase "goGL/internal/domain/purchase"
)

type Handler struct {
	svc purchase.Service
}

func NewHandler(svc purchase.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/purchase")

	// Supplier endpoints
	g.POST("/suppliers", h.createSupplier)
	g.GET("/suppliers", h.listSuppliers)
	g.GET("/suppliers/:id", h.getSupplier)
	g.PUT("/suppliers/:id", h.updateSupplier)
	g.DELETE("/suppliers/:id", h.deleteSupplier)
	g.GET("/suppliers/balance/:code", h.getSupplierBalance)

	// Purchase Order endpoints
	g.POST("/orders", h.createOrder)
	g.GET("/orders", h.listOrders)
	g.GET("/orders/:id", h.getOrder)
	g.PUT("/orders/:id", h.updateOrder)
	g.DELETE("/orders/:id", h.deleteOrder)
	g.POST("/orders/:id/confirm", h.confirmOrder)
	g.POST("/orders/:id/cancel", h.cancelOrder)

	// Goods Receipt endpoints
	g.POST("/receipts", h.createReceipt)
	g.GET("/receipts", h.listReceipts)
	g.GET("/receipts/:id", h.getReceipt)
	g.POST("/receipts/:id/approve", h.approveReceipt)

	// Purchase Invoice endpoints
	g.POST("/invoices", h.createInvoice)
	g.GET("/invoices", h.listInvoices)
	g.GET("/invoices/:id", h.getInvoice)
	g.PUT("/invoices/:id", h.updateInvoice)
	g.DELETE("/invoices/:id", h.deleteInvoice)
	g.POST("/invoices/:id/post", h.postInvoice)

	// Payment endpoints
	g.POST("/payments", h.createPayment)
	g.GET("/payments", h.listPayments)
	g.GET("/payments/:id", h.getPayment)
	g.POST("/payments/:id/approve", h.approvePayment)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	var ve *core.ValidationError
	if errors.As(err, &ve) {
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message, "field": ve.Field})
		return
	}
	switch {
	case errors.Is(err, domainpurchase.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domainpurchase.ErrInvalidStatus):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domainpurchase.ErrEmptyLines):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// --- Supplier Handlers ---

func (h *Handler) createSupplier(c *gin.Context) {
	var s domainpurchase.Supplier
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.CreateSupplier(c.Request.Context(), &s, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, got)
}

func (h *Handler) getSupplier(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.GetSupplier(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) updateSupplier(c *gin.Context) {
	id := c.Param("id")
	var s domainpurchase.Supplier
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.UpdateSupplier(c.Request.Context(), id, &s, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) deleteSupplier(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteSupplier(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (h *Handler) listSuppliers(c *gin.Context) {
	name := c.Query("name")
	status := domainpurchase.SupplierStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	got, err := h.svc.ListSuppliers(c.Request.Context(), name, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": got, "total": len(got)})
}

func (h *Handler) getSupplierBalance(c *gin.Context) {
	code := c.Param("code")
	balance, err := h.svc.GetSupplierBalance(c.Request.Context(), code)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, balance)
}

// --- Purchase Order Handlers ---

func (h *Handler) createOrder(c *gin.Context) {
	var o domainpurchase.PurchaseOrder
	if err := c.ShouldBindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.CreateOrder(c.Request.Context(), &o, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, got)
}

func (h *Handler) getOrder(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.GetOrder(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) updateOrder(c *gin.Context) {
	id := c.Param("id")
	var o domainpurchase.PurchaseOrder
	if err := c.ShouldBindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.UpdateOrder(c.Request.Context(), id, &o, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) deleteOrder(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteOrder(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (h *Handler) listOrders(c *gin.Context) {
	supplierCode := c.Query("supplier_code")
	status := domainpurchase.OrderStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	got, err := h.svc.ListOrders(c.Request.Context(), supplierCode, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": got, "total": len(got)})
}

func (h *Handler) confirmOrder(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.ConfirmOrder(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) cancelOrder(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Reason = ""
	}
	got, err := h.svc.CancelOrder(c.Request.Context(), id, body.Reason, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

// --- Goods Receipt Handlers ---

func (h *Handler) createReceipt(c *gin.Context) {
	var g domainpurchase.GoodsReceipt
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.CreateReceipt(c.Request.Context(), &g, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, got)
}

func (h *Handler) getReceipt(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.GetReceipt(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) approveReceipt(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.ApproveReceipt(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) listReceipts(c *gin.Context) {
	supplierCode := c.Query("supplier_code")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	got, err := h.svc.ListReceipts(c.Request.Context(), supplierCode, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": got, "total": len(got)})
}

// --- Purchase Invoice Handlers ---

func (h *Handler) createInvoice(c *gin.Context) {
	var inv domainpurchase.PurchaseInvoice
	if err := c.ShouldBindJSON(&inv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.CreateInvoice(c.Request.Context(), &inv, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, got)
}

func (h *Handler) getInvoice(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.GetInvoice(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) updateInvoice(c *gin.Context) {
	id := c.Param("id")
	var inv domainpurchase.PurchaseInvoice
	if err := c.ShouldBindJSON(&inv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.UpdateInvoice(c.Request.Context(), id, &inv, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) deleteInvoice(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteInvoice(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (h *Handler) listInvoices(c *gin.Context) {
	supplierCode := c.Query("supplier_code")
	status := domainpurchase.InvoiceStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	got, err := h.svc.ListInvoices(c.Request.Context(), supplierCode, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": got, "total": len(got)})
}

func (h *Handler) postInvoice(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.PostInvoice(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

// --- Payment Handlers ---

func (h *Handler) createPayment(c *gin.Context) {
	var p domainpurchase.Payment
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	got, err := h.svc.CreatePayment(c.Request.Context(), &p, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, got)
}

func (h *Handler) getPayment(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.GetPayment(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) approvePayment(c *gin.Context) {
	id := c.Param("id")
	got, err := h.svc.ApprovePayment(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, got)
}

func (h *Handler) listPayments(c *gin.Context) {
	supplierCode := c.Query("supplier_code")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	got, err := h.svc.ListPayments(c.Request.Context(), supplierCode, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": got, "total": len(got)})
}
