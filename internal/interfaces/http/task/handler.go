package task

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apptask "goGL/internal/application/task"
	"goGL/internal/domain/task"
)

type Handler struct {
	svc            apptask.Service
	identityHeader string
}

func NewHandler(svc apptask.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/tasks")
	g.GET("", h.listTasks)
	g.GET("/runs", h.listRuns)
	g.POST("/:name/run", h.runNow)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

func (h *Handler) listTasks(c *gin.Context) {
	tasks, err := h.svc.ListTasks(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) listRuns(c *gin.Context) {
	runs, err := h.svc.ListRuns(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handler) runNow(c *gin.Context) {
	run, err := h.svc.RunNow(c.Request.Context(), h.actor(c), c.Param("name"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

func respondError(c *gin.Context, err error) {
	code := http.StatusBadRequest
	switch {
	case errors.Is(err, task.ErrUnknown):
		code = http.StatusNotFound
	case errors.Is(err, task.ErrInProgress):
		code = http.StatusConflict
	}
	c.JSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}
