package authz

import (
	"net/http"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

// Handler exposes casbin policy management and an enforcement check. It talks
// to the shared enforcer directly; like the middleware it is cross-cutting
// infrastructure rather than a feature module, so it intentionally has no
// service/repository layers of its own.
type Handler struct {
	e *casbin.Enforcer
}

func NewHandler(e *casbin.Enforcer) *Handler {
	return &Handler{e: e}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/authz")
	g.GET("/policies", h.listPolicies)
	g.POST("/policies", h.addPolicy)
	g.DELETE("/policies", h.removePolicy)
	g.POST("/check", h.check)
}

type policyRequest struct {
	PType string   `json:"ptype"` // "p" for permission rules, "g" for role links
	Rule  []string `json:"rule"`
}

type checkRequest struct {
	Sub string `json:"sub"` // subject (user id or role)
	Obj string `json:"obj"` // object (route pattern, e.g. /api/v1/cash/*)
	Act string `json:"act"` // action (HTTP method or "*")
}

func (h *Handler) listPolicies(c *gin.Context) {
	p, err := h.e.GetPolicy()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	g, err := h.e.GetGroupingPolicy()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"p": p, "g": g})
}

func (h *Handler) addPolicy(c *gin.Context) {
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Rule) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ptype and a non-empty rule are required"})
		return
	}

	var (
		added bool
		err   error
	)
	switch req.PType {
	case "p":
		added, err = h.e.AddPolicy(req.Rule)
	case "g":
		added, err = h.e.AddGroupingPolicy(req.Rule)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "ptype must be \"p\" or \"g\""})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	status := http.StatusCreated
	if !added {
		status = http.StatusOK // already present
	}
	c.JSON(status, gin.H{"added": added})
}

func (h *Handler) removePolicy(c *gin.Context) {
	var req policyRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Rule) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ptype and a non-empty rule are required"})
		return
	}

	var (
		removed bool
		err     error
	)
	switch req.PType {
	case "p":
		removed, err = h.e.RemovePolicy(req.Rule)
	case "g":
		removed, err = h.e.RemoveGroupingPolicy(req.Rule)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "ptype must be \"p\" or \"g\""})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": removed})
}

func (h *Handler) check(c *gin.Context) {
	var req checkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sub, obj and act are required"})
		return
	}
	allowed, err := h.e.Enforce(req.Sub, req.Obj, req.Act)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"allowed": allowed})
}
