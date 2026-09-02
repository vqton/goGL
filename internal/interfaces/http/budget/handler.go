package budget

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/budget"
	domainbudget "goGL/internal/domain/budget"
	"goGL/internal/domain/core"
)

type Handler struct {
	svc budget.Service
}

func NewHandler(svc budget.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/budget")
	g.POST("/plans", h.createPlan)
	g.GET("/plans", h.listPlans)
	g.GET("/plans/:id", h.getPlan)
	g.PUT("/plans/:id", h.updatePlan)
	g.POST("/plans/:id/approve", h.approvePlan)
	g.POST("/plans/:id/lock", h.lockPlan)
	g.DELETE("/plans/:id", h.deletePlan)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainbudget.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domainbudget.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domainbudget.ErrLocked):
		c.JSON(http.StatusConflict, gin.H{"error": "plan is locked"})
	case errors.Is(err, domainbudget.ErrInvalid):
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

func (h *Handler) createPlan(c *gin.Context) {
	var input domainbudget.BudgetPlan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	plan, err := h.svc.Create(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": plan})
}

func (h *Handler) getPlan(c *gin.Context) {
	plan, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": plan})
}

func (h *Handler) updatePlan(c *gin.Context) {
	var patch domainbudget.BudgetPlan
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	plan, err := h.svc.Update(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": plan})
}

func (h *Handler) listPlans(c *gin.Context) {
	fiscalYear := 0
	if fy := c.Query("fiscal_year"); fy != "" {
		v, err := strconv.Atoi(fy)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fiscal_year"})
			return
		}
		fiscalYear = v
	}
	plans, err := h.svc.List(c.Request.Context(), fiscalYear, c.Query("department"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": plans})
}

func (h *Handler) approvePlan(c *gin.Context) {
	plan, err := h.svc.Approve(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": plan})
}

func (h *Handler) lockPlan(c *gin.Context) {
	plan, err := h.svc.Lock(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": plan})
}

func (h *Handler) deletePlan(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
