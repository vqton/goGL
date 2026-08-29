package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appuser "goGL/internal/application/user"
	"goGL/internal/domain/user"
)

type Handler struct {
	svc            appuser.Service
	identityHeader string
}

func NewHandler(svc appuser.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/users")
	g.GET("", h.listUsers)
	g.POST("", h.createUser)
	g.GET("/:id", h.getUser)
	g.PUT("/:id", h.updateUser)
	g.DELETE("/:id", h.deleteUser)
	g.POST("/:id/suspend", h.suspendUser)
	g.POST("/:id/activate", h.activateUser)
	g.POST("/:id/reset-password", h.resetPassword)

	roles := r.Group("/roles")
	roles.GET("", h.listRoles)
	roles.POST("", h.createRole)
	roles.PUT("/:code", h.updateRole)
	roles.DELETE("/:code", h.deleteRole)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

type createUserRequest struct {
	Username   string   `json:"username"`
	FullName   string   `json:"full_name"`
	Password   string   `json:"password"`
	RoleCodes  []string `json:"role_codes"`
}

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	u := &user.User{Username: req.Username, FullName: req.FullName, RoleCodes: req.RoleCodes}
	if err := h.svc.CreateUser(c.Request.Context(), h.actor(c), u, req.Password); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, u)
}

func (h *Handler) listUsers(c *gin.Context) {
	users, err := h.svc.ListUsers(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) getUser(c *gin.Context) {
	u, err := h.svc.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *Handler) updateUser(c *gin.Context) {
	var req user.User
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	req.ID = c.Param("id")
	if err := h.svc.UpdateUser(c.Request.Context(), h.actor(c), &req); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) deleteUser(c *gin.Context) {
	if err := h.svc.DeleteUser(c.Request.Context(), h.actor(c), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) suspendUser(c *gin.Context) {
	if err := h.svc.SuspendUser(c.Request.Context(), h.actor(c), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) activateUser(c *gin.Context) {
	if err := h.svc.ActivateUser(c.Request.Context(), h.actor(c), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type resetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func (h *Handler) resetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "new_password is required"}})
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), h.actor(c), c.Param("id"), req.NewPassword); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, roles)
}

func (h *Handler) createRole(c *gin.Context) {
	var role user.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	if err := h.svc.CreateRole(c.Request.Context(), h.actor(c), &role); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, role)
}

func (h *Handler) updateRole(c *gin.Context) {
	var role user.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	role.Code = c.Param("code")
	if err := h.svc.UpdateRole(c.Request.Context(), h.actor(c), &role); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, role)
}

func (h *Handler) deleteRole(c *gin.Context) {
	if err := h.svc.DeleteRole(c.Request.Context(), h.actor(c), c.Param("code")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondError(c *gin.Context, err error) {
	var code int
	switch {
	case errors.Is(err, user.ErrNotFound), errors.Is(err, user.ErrRoleNotFound):
		code = http.StatusNotFound
	case errors.Is(err, user.ErrLastAdmin),
		errors.Is(err, user.ErrRoleSystemProtected),
		errors.Is(err, user.ErrRoleInUse):
		code = http.StatusForbidden
	case errors.Is(err, user.ErrUsernameTaken),
		errors.Is(err, user.ErrRoleExists),
		errors.Is(err, user.ErrWeakPassword):
		code = http.StatusUnprocessableEntity
	default:
		code = http.StatusBadRequest
	}
	c.JSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}
