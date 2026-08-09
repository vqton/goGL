package system

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/system"
)

type Handler struct {
	svc system.Service
}

func NewHandler(svc system.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/system")
	g.POST("/tenants", h.createTenant)
	g.GET("/tenants/:id", h.getTenant)
}

func (h *Handler) createTenant(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getTenant(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
