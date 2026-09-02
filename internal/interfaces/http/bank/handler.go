package bank

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/bank"
	domainbank "goGL/internal/domain/bank"
	"goGL/internal/domain/core"
)

type Handler struct {
	svc bank.Service
}

func NewHandler(svc bank.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/bank")
	g.POST("/transactions", h.createTransaction)
	g.GET("/transactions", h.listTransactions)
	g.GET("/transactions/:id", h.getTransaction)
	g.PUT("/transactions/:id", h.updateTransaction)
	g.POST("/transactions/:id/clear", h.clearTransaction)
	g.POST("/transactions/:id/reconcile", h.reconcileTransaction)
	g.POST("/transactions/:id/cancel", h.cancelTransaction)
	g.DELETE("/transactions/:id", h.deleteTransaction)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainbank.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domainbank.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domainbank.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "state conflict"})
	case errors.Is(err, domainbank.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
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

func (h *Handler) createTransaction(c *gin.Context) {
	var input domainbank.BankTransaction
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Create(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tx})
}

func (h *Handler) getTransaction(c *gin.Context) {
	tx, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tx})
}

func (h *Handler) updateTransaction(c *gin.Context) {
	var patch domainbank.BankTransaction
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Update(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tx})
}

func (h *Handler) listTransactions(c *gin.Context) {
	txType := domainbank.TransactionType(c.Query("type"))
	txs, err := h.svc.List(c.Request.Context(), c.Query("account_no"), txType)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": txs})
}

func (h *Handler) clearTransaction(c *gin.Context) {
	tx, err := h.svc.Clear(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tx})
}

func (h *Handler) reconcileTransaction(c *gin.Context) {
	tx, err := h.svc.Reconcile(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tx})
}

func (h *Handler) cancelTransaction(c *gin.Context) {
	tx, err := h.svc.Cancel(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tx})
}

func (h *Handler) deleteTransaction(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
