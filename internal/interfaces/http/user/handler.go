package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/user"
)

type Handler struct {
	svc user.Service
}

func NewHandler(svc user.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/users")
	g.POST("/", h.createUser)
	g.GET("/:id", h.getUser)
	g.POST("/roles", h.assignRole)
}

func (h *Handler) createUser(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getUser(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) assignRole(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
