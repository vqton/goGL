package contract

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/contract"
)

type Handler struct {
	svc contract.Service
}

func NewHandler(svc contract.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/contracts")
	g.POST("/", h.createContract)
	g.GET("/:id", h.getContract)
	g.POST("/loans", h.createLoan)
}

func (h *Handler) createContract(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getContract(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) createLoan(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
