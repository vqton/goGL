package options

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/options"
)

type Handler struct {
	svc options.Service
}

func NewHandler(svc options.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/options")
	g.POST("/", h.setOption)
	g.GET("/:key", h.getOption)
}

func (h *Handler) setOption(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getOption(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
