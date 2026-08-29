package system

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appsystem "goGL/internal/application/system"
)

type Handler struct {
	svc appsystem.Service
}

func NewHandler(svc appsystem.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/system")
	g.GET("/info", h.info)
}

func (h *Handler) info(c *gin.Context) {
	info, err := h.svc.GetInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SYSTEM_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, info)
}
