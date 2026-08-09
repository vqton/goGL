package task

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/task"
)

type Handler struct {
	svc task.Service
}

func NewHandler(svc task.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/tasks")
	g.POST("/", h.createTask)
	g.GET("/:id", h.getTask)
}

func (h *Handler) createTask(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getTask(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
