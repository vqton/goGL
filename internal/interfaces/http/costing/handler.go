package costing

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/costing"
)

type Handler struct {
	svc costing.Service
}

func NewHandler(svc costing.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/costing")
	g.POST("/cost-sheets", h.createCostSheet)
	g.GET("/cost-sheets/:id", h.getCostSheet)
}

func (h *Handler) createCostSheet(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getCostSheet(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
