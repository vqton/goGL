package payroll

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/payroll"
)

type Handler struct {
	svc payroll.Service
}

func NewHandler(svc payroll.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/payroll")
	g.POST("/payslips", h.createPayslip)
	g.GET("/payslips/:id", h.getPayslip)
}

func (h *Handler) createPayslip(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getPayslip(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
