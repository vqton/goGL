package tools

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/tools"
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
	g.GET("/cards/:id", h.getCard)
}

func (h *Handler) createCard(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getCard(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
