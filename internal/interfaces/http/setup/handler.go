package setup

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	appsetup "goGL/internal/application/setup"
	domainsetup "goGL/internal/domain/setup"
)

type Handler struct {
	svc            appsetup.Service
	identityHeader string
}

func NewHandler(svc appsetup.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/setup")
	g.GET("/status", h.status)
	g.GET("/profile", h.getProfile)
	g.PUT("/profile", h.updateProfile)
	g.POST("/initialize", h.initialize)
	g.POST("/activate", h.activate)

	ob := g.Group("/opening-balances")
	ob.POST("", h.saveBalance)
	ob.GET("", h.listBalances)
	ob.DELETE("/:id", h.deleteBalance)
	ob.POST("/check", h.checkBalances)
	ob.POST("/import", h.importBalances)
	ob.GET("/import/:jobId/report", h.importReport)
	ob.GET("/import/:jobId/errors.csv", h.importErrorsCSV)
	ob.POST("/lock", h.lock)
	ob.POST("/reopen", h.reopen)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

func respondError(c *gin.Context, err error) {
	var code int
	switch {
	case errors.Is(err, domainsetup.ErrAlreadyInitialized),
		errors.Is(err, domainsetup.ErrWrongState),
		errors.Is(err, domainsetup.ErrBalanceLocked),
		errors.Is(err, domainsetup.ErrReopenBlocked),
		errors.Is(err, domainsetup.ErrUnbalanced):
		code = http.StatusConflict
	case errors.Is(err, domainsetup.ErrNotInitialized),
		errors.Is(err, domainsetup.ErrBalanceNotFound),
		errors.Is(err, domainsetup.ErrImportNotFound),
		errors.Is(err, sql.ErrNoRows):
		code = http.StatusNotFound
	case errors.Is(err, domainsetup.ErrInvalidProfile),
		errors.Is(err, domainsetup.ErrInvalidTaxCode),
		errors.Is(err, domainsetup.ErrInvalidFiscalYear),
		errors.Is(err, domainsetup.ErrInvalidCurrency),
		errors.Is(err, domainsetup.ErrInvalidRegime),
		errors.Is(err, domainsetup.ErrInvalidBalance),
		errors.Is(err, domainsetup.ErrAccountNotFound),
		errors.Is(err, domainsetup.ErrObjectRequired),
		errors.Is(err, domainsetup.ErrObjectNotFound),
		errors.Is(err, domainsetup.ErrInvalidImport):
		code = http.StatusUnprocessableEntity
	default:
		var ve *domainsetup.ValidationError
		if errors.As(err, &ve) {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"error":      ve.MessageEn,
				"message_vn": ve.MessageVn,
				"message_en": ve.MessageEn,
			})
			return
		}
		code = http.StatusBadRequest
	}
	c.AbortWithStatusJSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}

func (h *Handler) status(c *gin.Context) {
	view, err := h.svc.Status(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) getProfile(c *gin.Context) {
	p, err := h.svc.GetProfile(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) updateProfile(c *gin.Context) {
	var p domainsetup.CompanyProfile
	if err := c.ShouldBindJSON(&p); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	saved, err := h.svc.UpdateProfile(c.Request.Context(), &p, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, saved)
}

func (h *Handler) initialize(c *gin.Context) {
	var req appsetup.InitializeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	view, err := h.svc.Initialize(c.Request.Context(), &req, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) activate(c *gin.Context) {
	if err := h.svc.Activate(c.Request.Context(), h.actor(c)); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) saveBalance(c *gin.Context) {
	var b domainsetup.OpeningBalance
	if err := c.ShouldBindJSON(&b); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	saved, err := h.svc.SaveBalance(c.Request.Context(), &b, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, saved)
}

func (h *Handler) listBalances(c *gin.Context) {
	list, err := h.svc.ListBalances(c.Request.Context(), c.Query("account_code"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) deleteBalance(c *gin.Context) {
	if err := h.svc.DeleteBalance(c.Request.Context(), c.Param("id"), h.actor(c)); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) checkBalances(c *gin.Context) {
	check, err := h.svc.CheckBalances(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, check)
}

func (h *Handler) importBalances(c *gin.Context) {
	var body struct {
		DryRun bool       `json:"dry_run"`
		Rows   [][]string `json:"rows"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	res, err := h.svc.ImportBalances(c.Request.Context(), body.Rows, h.actor(c), body.DryRun)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// importReport returns the persisted per-row report for a previous import
// (spec §4: GET /opening-balances/import/:jobId/report). Dry-run and commit
// reports are both stored, so the operator can re-inspect errors later.
func (h *Handler) importReport(c *gin.Context) {
	job, err := h.svc.GetImportReport(c.Request.Context(), c.Param("jobId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

// importErrorsCSV exports the failed rows of a previous import as a CSV the
// operator can fix and re-upload (spec §5.4: "M errors row+reason").
func (h *Handler) importErrorsCSV(c *gin.Context) {
	job, err := h.svc.GetImportReport(c.Request.Context(), c.Param("jobId"))
	if err != nil {
		respondError(c, err)
		return
	}
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"row", "column", "message"})
	for _, e := range job.Errors {
		_ = w.Write([]string{strconv.Itoa(e.Row), e.Column, e.Message})
	}
	w.Flush()

	c.Header("Content-Disposition", "attachment; filename=opening-balances-import-errors-"+job.ID+".csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, sb.String())
}

func (h *Handler) lock(c *gin.Context) {
	if err := h.svc.Lock(c.Request.Context(), h.actor(c)); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) reopen(c *gin.Context) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	if err := h.svc.Reopen(c.Request.Context(), h.actor(c), body.Reason); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
