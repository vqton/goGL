package tools

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/tools"
	"goGL/internal/domain/core"
	domaintools "goGL/internal/domain/tools"
)

type Handler struct {
	svc tools.Service
}

func NewHandler(svc tools.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/tools")
	g.POST("/cards", h.createCard)
	g.GET("/cards", h.listCards)
	g.GET("/cards/:id", h.getCard)
	g.PUT("/cards/:id", h.updateCard)
	g.DELETE("/cards/:id", h.deleteCard)
	g.POST("/cards/:id/scrap", h.scrapCard)

	// Transaction endpoints
	g.POST("/cards/:id/import", h.importTool)
	g.POST("/cards/:id/export", h.exportTool)
	g.POST("/cards/:id/transfer", h.transferTool)
	g.POST("/cards/:id/return", h.returnTool)
	g.POST("/cards/:id/dispose", h.disposeTool)
	g.GET("/cards/:id/transactions", h.listTransactions)
	g.GET("/cards/:id/stock", h.getStock)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domaintools.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domaintools.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domaintools.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "state conflict"})
	case errors.Is(err, domaintools.ErrInvalid):
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

func (h *Handler) createCard(c *gin.Context) {
	var input domaintools.ToolCard
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	card, err := h.svc.Create(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": card})
}

func (h *Handler) getCard(c *gin.Context) {
	card, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func (h *Handler) updateCard(c *gin.Context) {
	var patch domaintools.ToolCard
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	card, err := h.svc.Update(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func (h *Handler) listCards(c *gin.Context) {
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	cards, err := h.svc.List(c.Request.Context(),
		c.Query("category"),
		domaintools.CardState(c.Query("state")),
		limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cards})
}

// --- Transaction handlers ---

func (h *Handler) importTool(c *gin.Context) {
	var input struct {
		Quantity  int    `json:"quantity"`
		UnitPrice int64  `json:"unit_price"`
		Reference string `json:"reference"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Import(c.Request.Context(), c.Param("id"), input.Quantity, input.UnitPrice, input.Reference, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tx})
}

func (h *Handler) exportTool(c *gin.Context) {
	var input struct {
		Quantity    int    `json:"quantity"`
		ToDepartment string `json:"to_department"`
		ToPerson    string `json:"to_person"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Export(c.Request.Context(), c.Param("id"), input.Quantity, input.ToDepartment, input.ToPerson, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tx})
}

func (h *Handler) transferTool(c *gin.Context) {
	var input struct {
		Quantity     int    `json:"quantity"`
		ToLocation   string `json:"to_location"`
		ToDepartment string `json:"to_department"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Transfer(c.Request.Context(), c.Param("id"), input.Quantity, input.ToLocation, input.ToDepartment, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tx})
}

func (h *Handler) returnTool(c *gin.Context) {
	var input struct {
		Quantity  int    `json:"quantity"`
		Reason    string `json:"reason"`
		Reference string `json:"reference"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Return(c.Request.Context(), c.Param("id"), input.Quantity, input.Reason, input.Reference, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tx})
}

func (h *Handler) disposeTool(c *gin.Context) {
	var input struct {
		Quantity int    `json:"quantity"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	tx, err := h.svc.Dispose(c.Request.Context(), c.Param("id"), input.Quantity, input.Reason, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tx})
}

func (h *Handler) listTransactions(c *gin.Context) {
	txType := domaintools.TransactionType(c.Query("type"))
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	tx, err := h.svc.ListTransactions(c.Request.Context(), c.Param("id"), txType, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tx})
}

func (h *Handler) getStock(c *gin.Context) {
	stock, err := h.svc.GetStock(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"stock": stock}})
}

func (h *Handler) scrapCard(c *gin.Context) {
	card, err := h.svc.Scrap(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": card})
}

func (h *Handler) deleteCard(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
