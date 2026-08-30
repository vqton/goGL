package sales

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/sales"
	"goGL/internal/domain/core"
	domainsales "goGL/internal/domain/sales"
)

type Handler struct {
	svc sales.Service
}

func NewHandler(svc sales.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/sales")

	// Invoice endpoints
	g.POST("/invoices", h.createInvoice)
	g.GET("/invoices", h.listInvoices)
	g.GET("/invoices/:id", h.getInvoice)
	g.PUT("/invoices/:id", h.updateInvoice)
	g.DELETE("/invoices/:id", h.deleteInvoice)

	// Order endpoints
	g.POST("/orders", h.createOrder)
	g.GET("/orders", h.listOrders)
	g.GET("/orders/:id", h.getOrder)
	g.PUT("/orders/:id", h.updateOrder)
	g.DELETE("/orders/:id", h.deleteOrder)
	g.POST("/orders/:id/confirm", h.confirmOrder)
	g.POST("/orders/:id/cancel", h.cancelOrder)

	// Return endpoints
	g.POST("/returns", h.createReturn)
	g.GET("/returns", h.listReturns)
	g.GET("/returns/:id", h.getReturn)
	g.POST("/returns/:id/approve", h.approveReturn)
	g.POST("/returns/:id/receive", h.receiveReturn)

	// Customer
	g.GET("/customers/:code/balance", h.getCustomerBalance)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainsales.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domainsales.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domainsales.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "state conflict"})
	case errors.Is(err, domainsales.ErrInvalid),
		errors.Is(err, domainsales.ErrEmptyLines),
		errors.Is(err, domainsales.ErrReturnExceedsQty),
		errors.Is(err, domainsales.ErrInvalidQuantity):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domainsales.ErrInsufficientQty):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "insufficient quantity"})
	default:
		var ve *core.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": ve.Message,
				"field": ve.Field,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// --- Invoice handlers ---

func (h *Handler) createInvoice(c *gin.Context) {
	var input domainsales.SalesInvoice
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.CreateInvoice(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) getInvoice(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.GetInvoice(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) updateInvoice(c *gin.Context) {
	id := c.Param("id")
	var input domainsales.SalesInvoice
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.UpdateInvoice(c.Request.Context(), id, &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) deleteInvoice(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteInvoice(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) listInvoices(c *gin.Context) {
	customerCode := c.Query("customer_code")
	status := domainsales.InvoiceStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	result, err := h.svc.ListInvoices(c.Request.Context(), customerCode, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// --- Order handlers ---

func (h *Handler) createOrder(c *gin.Context) {
	var input domainsales.SalesOrder
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.CreateOrder(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) getOrder(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.GetOrder(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) updateOrder(c *gin.Context) {
	id := c.Param("id")
	var input domainsales.SalesOrder
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.UpdateOrder(c.Request.Context(), id, &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) deleteOrder(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteOrder(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) listOrders(c *gin.Context) {
	customerCode := c.Query("customer_code")
	status := domainsales.OrderStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	result, err := h.svc.ListOrders(c.Request.Context(), customerCode, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) confirmOrder(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.ConfirmOrder(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) cancelOrder(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.CancelOrder(c.Request.Context(), id, input.Reason, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// --- Return handlers ---

func (h *Handler) createReturn(c *gin.Context) {
	var input domainsales.SalesReturn
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.CreateReturn(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) getReturn(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.GetReturn(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) listReturns(c *gin.Context) {
	customerCode := c.Query("customer_code")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	result, err := h.svc.ListReturns(c.Request.Context(), customerCode, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) approveReturn(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.ApproveReturn(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) receiveReturn(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.ReceiveReturn(c.Request.Context(), id, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// --- Customer handler ---

func (h *Handler) getCustomerBalance(c *gin.Context) {
	code := c.Param("code")
	balance, err := h.svc.GetCustomerBalance(c.Request.Context(), code)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"customer_code": code,
		"balance":       balance,
	})
}
