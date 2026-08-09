package backup

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/backup"
)

type Handler struct {
	svc backup.Service
}

func NewHandler(svc backup.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/backup")
	g.POST("/jobs", h.createJob)
	g.GET("/jobs/:id", h.getJob)
	g.POST("/jobs/:id/restore", h.restore)
}

func (h *Handler) createJob(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getJob(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) restore(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
