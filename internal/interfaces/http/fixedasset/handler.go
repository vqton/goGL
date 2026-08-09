package fixedasset

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/fixedasset"
)

type Handler struct {
	svc fixedasset.Service
}

func NewHandler(svc fixedasset.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/fixed-assets")
	g.POST("/", h.createAsset)
	g.GET("/:id", h.getAsset)
}

func (h *Handler) createAsset(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getAsset(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
