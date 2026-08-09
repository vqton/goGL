package tax

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/tax"
)

type Handler struct {
	svc tax.Service
}

func NewHandler(svc tax.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/tax")
	g.POST("/declarations", h.createDeclaration)
	g.GET("/declarations/:id", h.getDeclaration)
}

func (h *Handler) createDeclaration(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getDeclaration(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
