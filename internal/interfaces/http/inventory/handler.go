package inventory

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/inventory"
)

type Handler struct {
	svc inventory.Service
}

func NewHandler(svc inventory.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/inventory")
	g.POST("/movements", h.createMovement)
	g.GET("/movements/:id", h.getMovement)
}

func (h *Handler) createMovement(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getMovement(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
