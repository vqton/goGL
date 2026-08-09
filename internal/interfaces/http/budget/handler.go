package budget

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/budget"
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
	g.GET("/plans/:id", h.getPlan)
}

func (h *Handler) createPlan(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getPlan(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
