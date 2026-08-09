package document

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/document"
)

type Handler struct {
	svc document.Service
}

func NewHandler(svc document.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/documents")
	g.POST("/", h.createDocument)
	g.GET("/:id", h.getDocument)
}

func (h *Handler) createDocument(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getDocument(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
