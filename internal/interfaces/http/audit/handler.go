package audit

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/audit"
)

type Handler struct {
	svc audit.Service
}

func NewHandler(svc audit.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/audit")
	g.POST("/logs", h.record)
	g.GET("/logs/:id", h.getLog)
}

func (h *Handler) record(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getLog(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
