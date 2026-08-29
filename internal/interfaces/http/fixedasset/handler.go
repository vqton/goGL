package fixedasset

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/fixedasset"
	"goGL/internal/domain/core"
	dom "goGL/internal/domain/fixedasset"
)

type Handler struct {
	svc fixedasset.Service
}

func NewHandler(svc fixedasset.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/fixed-assets")
	g.POST("", h.Create)
	g.GET("/:id", h.GetByID)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.GET("", h.List)
	g.POST("/:id/transfer", h.Transfer)
	g.POST("/:id/liquidate", h.Liquidate)
	g.POST("/:id/confirm-liquidation", h.ConfirmLiquidation)
	g.POST("/:id/deactivate", h.Deactivate)
	g.POST("/:id/reactivate", h.Reactivate)

	// Depreciation endpoints
	g.POST("/depreciation/calculate", h.RunMonthlyDepreciation)
	g.GET("/depreciation/period/:period", h.GetDepreciationByPeriod)
	g.GET("/:id/depreciation-schedule", h.GetDepreciationSchedule)

	// Approval workflow endpoints
	g.POST("/:id/approve-liquidation", h.ApproveLiquidation)
	g.POST("/:id/reject-liquidation", h.RejectLiquidation)
}

func (h *Handler) Create(c *gin.Context) {
	var a dom.FixedAsset
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Create(c.Request.Context(), &a); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	a, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var a dom.FixedAsset
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a.ID = id
	if err := h.svc.Update(c.Request.Context(), &a); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	assetType := dom.AssetType(c.Query("asset_type"))
	state := dom.AssetState(c.Query("state"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	list, err := h.svc.List(c.Request.Context(), assetType, state, limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) Transfer(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Location   string `json:"location"`
		Department string `json:"department"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.Transfer(c.Request.Context(), id, req.Location, req.Department)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) Liquidate(c *gin.Context) {
	id := c.Param("id")
	a, err := h.svc.Liquidate(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) ConfirmLiquidation(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		State dom.AssetState `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.ConfirmLiquidation(c.Request.Context(), id, req.State)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) Deactivate(c *gin.Context) {
	id := c.Param("id")
	a, err := h.svc.Deactivate(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) Reactivate(c *gin.Context) {
	id := c.Param("id")
	a, err := h.svc.Reactivate(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) RunMonthlyDepreciation(c *gin.Context) {
	var req struct {
		Period string `json:"period" binding:"required"`
		Actor  string `json:"actor" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entries, err := h.svc.RunMonthlyDepreciation(c.Request.Context(), req.Period, req.Actor)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"period":   req.Period,
		"count":    len(entries),
		"entries":  entries,
	})
}

func (h *Handler) GetDepreciationByPeriod(c *gin.Context) {
	period := c.Param("period")
	entries, err := h.svc.GetDepreciationByPeriod(c.Request.Context(), period)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"period":  period,
		"count":   len(entries),
		"entries": entries,
	})
}

func (h *Handler) GetDepreciationSchedule(c *gin.Context) {
	id := c.Param("id")
	entries, err := h.svc.GetDepreciationSchedule(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"asset_id": id,
		"count":    len(entries),
		"entries":  entries,
	})
}

func (h *Handler) ApproveLiquidation(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Actor string `json:"actor" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.ApproveLiquidation(c.Request.Context(), id, req.Actor)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) RejectLiquidation(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Actor  string `json:"actor" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.RejectLiquidation(c.Request.Context(), id, req.Actor, req.Reason)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) handleError(c *gin.Context, err error) {
	var ve *core.ValidationError
	if errors.As(err, &ve) {
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
		return
	}
	switch {
	case errors.Is(err, dom.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, dom.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, dom.ErrLiquidationPending):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, dom.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
