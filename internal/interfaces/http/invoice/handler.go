package invoice

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/invoice"
)

type Handler struct {
	svc invoice.Service
}

func NewHandler(svc invoice.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/invoice")
	g.POST("/", h.createInvoice)
	g.GET("/:id", h.getInvoice)
}

func (h *Handler) createInvoice(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getInvoice(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
