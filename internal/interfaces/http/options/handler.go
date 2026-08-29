package options

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appoptions "goGL/internal/application/options"
	"goGL/internal/domain/options"
)

type Handler struct {
	svc            appoptions.Service
	identityHeader string
}

func NewHandler(svc appoptions.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/options")
	g.GET("", h.listOptions)
	g.GET("/:key", h.getOption)
	g.PUT("/:key", h.setOption)
	g.POST("/:key/reset", h.resetOption)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

func (h *Handler) listOptions(c *gin.Context) {
	opts, err := h.svc.ListOptions(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, opts)
}

func (h *Handler) getOption(c *gin.Context) {
	o, err := h.svc.GetOption(c.Request.Context(), c.Param("key"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *Handler) setOption(c *gin.Context) {
	var o options.Option
	if err := c.ShouldBindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	o.Key = c.Param("key")
	if err := h.svc.SetOption(c.Request.Context(), h.actor(c), &o); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *Handler) resetOption(c *gin.Context) {
	if err := h.svc.ResetOption(c.Request.Context(), h.actor(c), c.Param("key")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondError(c *gin.Context, err error) {
	code := http.StatusBadRequest
	if errors.Is(err, options.ErrNotFound) {
		code = http.StatusNotFound
	} else if errors.Is(err, options.ErrInvalidValue) {
		code = http.StatusUnprocessableEntity
	}
	c.JSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}
