package document

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/document"
	"goGL/internal/domain/core"
	domaindoc "goGL/internal/domain/document"
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
	g.GET("/", h.listDocuments)
	g.GET("/:id", h.getDocument)
	g.PUT("/:id", h.updateDocument)
	g.POST("/:id/archive", h.archiveDocument)
	g.DELETE("/:id", h.deleteDocument)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domaindoc.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domaindoc.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domaindoc.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, domaindoc.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	default:
		var ve *core.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   ve.Message,
				"field":   ve.Field,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) createDocument(c *gin.Context) {
	var input domaindoc.Document
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	doc, err := h.svc.Create(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": doc})
}

func (h *Handler) getDocument(c *gin.Context) {
	doc, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *Handler) updateDocument(c *gin.Context) {
	var patch domaindoc.Document
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	doc, err := h.svc.Update(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *Handler) listDocuments(c *gin.Context) {
	docs, err := h.svc.List(c.Request.Context(),
		c.Query("owner"), c.Query("type"), c.Query("state"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (h *Handler) archiveDocument(c *gin.Context) {
	doc, err := h.svc.Archive(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *Handler) deleteDocument(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id"), h.actor(c)); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
