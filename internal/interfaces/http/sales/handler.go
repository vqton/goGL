package sales

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/sales"
)

type Handler struct {
	svc sales.Service
}

func NewHandler(svc sales.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/sales")
	g.POST("/invoices", h.createInvoice)
	g.GET("/invoices/:id", h.getInvoice)
}

func (h *Handler) createInvoice(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getInvoice(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
