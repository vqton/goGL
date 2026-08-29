package audit

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/audit"
	domainaudit "goGL/internal/domain/audit"
)

type Handler struct {
	svc audit.Service
}

func NewHandler(svc audit.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/audit")
	g.POST("/logs", h.record)
	g.GET("/logs", h.listRecent)
	g.GET("/logs/:id", h.getLog)
}

func (h *Handler) record(c *gin.Context) {
	var l domainaudit.AuditLog
	if err := c.ShouldBindJSON(&l); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	if err := h.svc.Record(c.Request.Context(), &l); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "AUDIT_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, l)
}

func (h *Handler) listRecent(c *gin.Context) {
	limit := 50
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	logs, err := h.svc.ListRecent(c.Request.Context(), c.Query("module"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "AUDIT_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) getLog(c *gin.Context) {
	l, err := h.svc.GetLog(c.Request.Context(), c.Param("id"))
	if errors.Is(err, domainaudit.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "audit log not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "AUDIT_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, l)
}
