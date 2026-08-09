package masterdata

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/masterdata"
)

type Handler struct {
	svc masterdata.Service
}

func NewHandler(svc masterdata.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/master-data")
	g.POST("/items", h.createItem)
	g.GET("/items/:id", h.getItem)
}

func (h *Handler) createItem(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getItem(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
