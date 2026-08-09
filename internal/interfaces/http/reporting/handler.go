package reporting

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/reporting"
)

type Handler struct {
	svc reporting.Service
}

func NewHandler(svc reporting.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/reporting")
	g.POST("/reports", h.createReport)
	g.GET("/reports/:id", h.getReport)
}

func (h *Handler) createReport(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getReport(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
