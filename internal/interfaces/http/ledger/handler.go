package ledger

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/ledger"
)

type Handler struct {
	svc ledger.Service
}

func NewHandler(svc ledger.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/ledger")
	g.POST("/entries", h.createEntry)
	g.GET("/entries/:id", h.getEntry)
}

func (h *Handler) createEntry(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getEntry(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
