package contract

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/contract"
	"goGL/internal/domain/core"
	domaincontract "goGL/internal/domain/contract"
)

type Handler struct {
	svc contract.Service
}

func NewHandler(svc contract.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/contracts")
	g.POST("/", h.createContract)
	g.GET("/", h.listContracts)
	g.GET("/:id", h.getContract)
	g.PUT("/:id", h.updateContract)
	g.POST("/:id/activate", h.activateContract)
	g.POST("/:id/terminate", h.terminateContract)
	g.DELETE("/:id", h.deleteContract)

	g.POST("/loans", h.createLoan)
	g.GET("/loans/:id", h.getLoan)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domaincontract.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domaincontract.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate"})
	case errors.Is(err, domaincontract.ErrLocked):
		c.JSON(http.StatusConflict, gin.H{"error": "contract is locked"})
	case errors.Is(err, domaincontract.ErrInvalid):
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

func (h *Handler) createContract(c *gin.Context) {
	var input domaincontract.Contract
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.Create(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *Handler) getContract(c *gin.Context) {
	result, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) updateContract(c *gin.Context) {
	var patch domaincontract.Contract
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.Update(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) listContracts(c *gin.Context) {
	results, err := h.svc.List(c.Request.Context(),
		domaincontract.ContractType(c.Query("type")),
		domaincontract.ContractState(c.Query("state")))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}

func (h *Handler) activateContract(c *gin.Context) {
	result, err := h.svc.Activate(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) terminateContract(c *gin.Context) {
	result, err := h.svc.Terminate(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) deleteContract(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) createLoan(c *gin.Context) {
	var input domaincontract.LoanAgreement
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	result, err := h.svc.CreateLoan(c.Request.Context(), &input, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *Handler) getLoan(c *gin.Context) {
	result, err := h.svc.GetLoan(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
