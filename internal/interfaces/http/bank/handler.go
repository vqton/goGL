package bank

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/bank"
)

type Handler struct {
	svc bank.Service
}

func NewHandler(svc bank.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/bank")
	g.POST("/transactions", h.createTransaction)
	g.GET("/transactions/:id", h.getTransaction)
}

func (h *Handler) createTransaction(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getTransaction(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
