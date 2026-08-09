package setup

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/setup"
)

type Handler struct {
	svc setup.Service
}

func NewHandler(svc setup.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/setup")
	g.POST("/initialize", h.initialize)
	g.GET("/profile/:id", h.getProfile)
	g.POST("/opening-balances", h.importOpeningBalances)
}

func (h *Handler) initialize(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) getProfile(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}

func (h *Handler) importOpeningBalances(c *gin.Context) {
	// TODO: implement
	c.Status(http.StatusNotImplemented)
}
