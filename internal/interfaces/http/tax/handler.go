package tax

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/tax"
	"goGL/internal/domain/core"
	domaintax "goGL/internal/domain/tax"
)

type Handler struct {
	svc tax.Service
}

func NewHandler(svc tax.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/tax")
	g.POST("/declarations", h.createDeclaration)
	g.GET("/declarations", h.listDeclarations)
	g.GET("/declarations/:id", h.getDeclaration)
	g.PUT("/declarations/:id", h.updateDeclaration)
	g.POST("/declarations/:id/file", h.fileDeclaration)
	g.DELETE("/declarations/:id", h.deleteDeclaration)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domaintax.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domaintax.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domaintax.ErrLocked):
		c.JSON(http.StatusConflict, gin.H{"error": "declaration is locked"})
	case errors.Is(err, domaintax.ErrInvalid):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		var ve *core.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": ve.Message,
				"field": ve.Field,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) createDeclaration(c *gin.Context) {
	var input domaintax.TaxDeclaration
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	decl, err := h.svc.Create(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": decl})
}

func (h *Handler) getDeclaration(c *gin.Context) {
	decl, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": decl})
}

func (h *Handler) updateDeclaration(c *gin.Context) {
	var patch domaintax.TaxDeclaration
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	decl, err := h.svc.Update(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": decl})
}

func (h *Handler) listDeclarations(c *gin.Context) {
	decls, err := h.svc.List(c.Request.Context(),
		domaintax.TaxType(c.Query("type")), c.Query("period"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": decls})
}

func (h *Handler) fileDeclaration(c *gin.Context) {
	decl, err := h.svc.File(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": decl})
}

func (h *Handler) deleteDeclaration(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
