package backup

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appbackup "goGL/internal/application/backup"
	"goGL/internal/domain/backup"
)

type Handler struct {
	svc            appbackup.Service
	identityHeader string
}

func NewHandler(svc appbackup.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/backup")
	g.POST("", h.createBackup)
	g.GET("", h.listBackups)
	g.DELETE("/:id", h.deleteBackup)
	g.POST("/:id/restore/stage", h.stageRestore)
	g.POST("/restore/:plan/approve", h.approveRestore)
	g.GET("/job", h.getJob)
	g.PUT("/job", h.setJob)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

type createBackupRequest struct {
	Tier    string `json:"tier"`
	Trigger string `json:"trigger"`
}

func (h *Handler) createBackup(c *gin.Context) {
	var req createBackupRequest
	_ = c.ShouldBindJSON(&req)
	a, err := h.svc.CreateBackup(c.Request.Context(), h.actor(c), req.Tier, req.Trigger)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) listBackups(c *gin.Context) {
	arts, err := h.svc.ListBackups(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, arts)
}

func (h *Handler) deleteBackup(c *gin.Context) {
	if err := h.svc.DeleteBackup(c.Request.Context(), h.actor(c), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) stageRestore(c *gin.Context) {
	plan, err := h.svc.StageRestore(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handler) approveRestore(c *gin.Context) {
	if err := h.svc.ApproveRestore(c.Request.Context(), h.actor(c), c.Param("plan")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) getJob(c *gin.Context) {
	j, err := h.svc.GetJob(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, j)
}

func (h *Handler) setJob(c *gin.Context) {
	var j backup.BackupJob
	if err := c.ShouldBindJSON(&j); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	out, err := h.svc.SetJob(c.Request.Context(), h.actor(c), &j)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func respondError(c *gin.Context, err error) {
	code := http.StatusBadRequest
	switch {
	case errors.Is(err, backup.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, backup.ErrIntegrity):
		code = http.StatusUnprocessableEntity
	case errors.Is(err, backup.ErrNoActivePlan):
		code = http.StatusConflict
	}
	c.JSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}
