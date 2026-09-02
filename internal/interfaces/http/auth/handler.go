package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appauth "goGL/internal/application/auth"
	"goGL/internal/domain/user"
)

type Handler struct {
	svc        appauth.Service
	cookieName string
	maxHours   int
}

func NewHandler(svc appauth.Service, cookieName string, maxHours int) *Handler {
	return &Handler{svc: svc, cookieName: cookieName, maxHours: maxHours}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/auth")
	g.POST("/login", h.login)
	g.POST("/logout", h.logout)
	g.POST("/change-password", h.changePassword)
	g.GET("/me", h.me)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "username and password are required"}})
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	ses, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, user.ErrLockedOut) {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": gin.H{"code": "AUTH_FAILED", "message": err.Error()}})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookieName,
		Value:    ses.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   h.maxHours * 3600,
	})
	c.JSON(http.StatusOK, gin.H{
		"session": gin.H{
			"id":         ses.ID,
			"username":   ses.Username,
			"expires_at": ses.ExpiresAt,
		},
	})
}

func (h *Handler) logout(c *gin.Context) {
	if token, err := c.Cookie(h.cookieName); err == nil && token != "" {
		_ = h.svc.Logout(c.Request.Context(), token)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	c.Status(http.StatusNoContent)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) changePassword(c *gin.Context) {
	actor, ok := h.currentUser(c)
	if !ok {
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "current_password and new_password are required"}})
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), actor.ID, req.CurrentPassword, req.NewPassword); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) me(c *gin.Context) {
	actor, ok := h.currentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, actor)
}

// currentUser resolves the session cookie; the auth group is mounted outside
// the Casbin middleware, so these endpoints enforce the session themselves.
func (h *Handler) currentUser(c *gin.Context) (*user.User, bool) {
	token, err := c.Cookie(h.cookieName)
	if err != nil || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "missing session"}})
		return nil, false
	}
	u, err := h.svc.Validate(c.Request.Context(), token)
	if err != nil {
		http.SetCookie(c.Writer, &http.Cookie{
			Name: h.cookieName, Value: "", Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode,
			Expires: time.Unix(0, 0),
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired session"}})
		return nil, false
	}
	return u, true
}

func (h *Handler) fail(c *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, user.ErrInvalidCredentials), errors.Is(err, user.ErrInvalidSession), errors.Is(err, user.ErrSuspended):
		status = http.StatusUnauthorized
	case errors.Is(err, user.ErrWeakPassword):
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, gin.H{"error": gin.H{"code": "AUTH_FAILED", "message": err.Error()}})
}
