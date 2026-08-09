package cash

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	appcash "goGL/internal/application/cash"
	domaincash "goGL/internal/domain/cash"
)

type Handler struct {
	svc            appcash.Service
	identityHeader string
}

func NewHandler(svc appcash.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/cash")
	g.POST("/funds", h.createFund)
	g.GET("/funds", h.listFunds)
	g.POST("/vouchers", h.createVoucher)
	g.GET("/vouchers", h.listVouchers)
	g.GET("/vouchers/:id", h.getVoucher)
	g.PATCH("/vouchers/:id", h.updateVoucher)
	g.POST("/vouchers/:id/approve", h.approveVoucher)
	g.POST("/vouchers/:id/post", h.postVoucher)
	g.POST("/vouchers/:id/void", h.voidVoucher)
	g.GET("/book", h.getCashBook)
	g.POST("/close-day", h.closeDay)
	g.GET("/counts", h.listCashCounts)
	g.POST("/counts", h.createCashCount)
	g.POST("/counts/:id/resolve", h.resolveCashCount)
	g.POST("/reconcile", h.reconcileMonth)
	g.GET("/reconciliations", h.listReconciliations)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

func respondError(c *gin.Context, err error) {
	var code int
	switch {
	case errors.Is(err, sql.ErrNoRows),
		errors.Is(err, domaincash.ErrNotFound),
		errors.Is(err, domaincash.ErrFundNotFound),
		errors.Is(err, domaincash.ErrVoucherNotFound):
		code = http.StatusNotFound
	case errors.Is(err, domaincash.ErrSelfApproval),
		errors.Is(err, domaincash.ErrCashierRequired),
		errors.Is(err, domaincash.ErrUnauthorizedActor):
		code = http.StatusForbidden
	case errors.Is(err, domaincash.ErrWrongState):
		code = http.StatusConflict
	case errors.Is(err, domaincash.ErrInvalidLines),
		errors.Is(err, domaincash.ErrFundInactive),
		errors.Is(err, domaincash.ErrPeriodClosed),
		errors.Is(err, domaincash.ErrNegativeBalance),
		errors.Is(err, domaincash.ErrOpenCountPending),
		errors.Is(err, domaincash.ErrReversalMissing),
		errors.Is(err, domaincash.ErrReversalMismatch),
		errors.Is(err, domaincash.ErrInvalidSigners),
		errors.Is(err, domaincash.ErrInvalidCount),
		errors.Is(err, domaincash.ErrUnpostedVouchers):
		code = http.StatusUnprocessableEntity
	default:
		code = http.StatusBadRequest
	}
	c.AbortWithStatusJSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}

func (h *Handler) createFund(c *gin.Context) {
	var f domaincash.Fund
	if err := c.ShouldBindJSON(&f); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	if err := h.svc.CreateFund(c.Request.Context(), &f); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, f)
}

func (h *Handler) listFunds(c *gin.Context) {
	funds, err := h.svc.ListFunds(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, funds)
}

func (h *Handler) createVoucher(c *gin.Context) {
	var v domaincash.Voucher
	if err := c.ShouldBindJSON(&v); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	if err := h.svc.CreateVoucher(c.Request.Context(), h.actor(c), &v); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) listVouchers(c *gin.Context) {
	filter := domaincash.VoucherFilter{
		FundID: c.Query("fund_id"),
		State:  domaincash.VoucherState(c.Query("state")),
		Type:   domaincash.VoucherType(c.Query("type")),
		From:   c.Query("from"),
		To:     c.Query("to"),
	}
	vouchers, err := h.svc.ListVouchers(c.Request.Context(), filter)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, vouchers)
}

func (h *Handler) getVoucher(c *gin.Context) {
	v, err := h.svc.GetVoucher(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) updateVoucher(c *gin.Context) {
	var v domaincash.Voucher
	if err := c.ShouldBindJSON(&v); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	v.ID = c.Param("id")
	if err := h.svc.UpdateVoucher(c.Request.Context(), h.actor(c), &v); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) approveVoucher(c *gin.Context) {
	v, err := h.svc.ApproveVoucher(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) postVoucher(c *gin.Context) {
	v, err := h.svc.PostVoucher(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) getCashBook(c *gin.Context) {
	entries, err := h.svc.GetCashBook(c.Request.Context(), c.Query("fund_id"), c.Query("from"), c.Query("to"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

type closeDayRequest struct {
	FundID        string   `json:"fund_id"`
	Date          string   `json:"date"`
	CountedAmount int64    `json:"counted_amount"`
	Participants  []string `json:"participants"`
}

func (h *Handler) closeDay(c *gin.Context) {
	var req closeDayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	count, err := h.svc.CloseDay(c.Request.Context(), h.actor(c), req.FundID, req.Date, req.CountedAmount, req.Participants)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, count)
}

func (h *Handler) listCashCounts(c *gin.Context) {
	counts, err := h.svc.ListCashCounts(c.Request.Context(), c.Query("fund_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, counts)
}

type createCountRequest struct {
	FundID        string   `json:"fund_id"`
	Date          string   `json:"date"`
	CountedAmount int64    `json:"counted_amount"`
	Participants  []string `json:"participants"`
}

func (h *Handler) createCashCount(c *gin.Context) {
	var req createCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	count, err := h.svc.CreateCashCount(c.Request.Context(), h.actor(c), req.FundID, req.Date, req.CountedAmount, req.Participants)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, count)
}

type resolveCountRequest struct {
	Resolution string `json:"resolution"`
}

func (h *Handler) resolveCashCount(c *gin.Context) {
	var req resolveCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	count, err := h.svc.ResolveCashCount(c.Request.Context(), h.actor(c), c.Param("id"), req.Resolution)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, count)
}

type voidRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) voidVoucher(c *gin.Context) {
	var req voidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	v, err := h.svc.VoidVoucher(c.Request.Context(), h.actor(c), c.Param("id"), req.Reason)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

type reconcileRequest struct {
	FundID            string   `json:"fund_id"`
	Period            string   `json:"period"`
	AccountantBalance int64    `json:"accountant_balance"`
	Signers           []string `json:"signers"`
}

func (h *Handler) reconcileMonth(c *gin.Context) {
	var req reconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	rec, err := h.svc.ReconcileMonth(c.Request.Context(), h.actor(c), req.FundID, req.Period, req.AccountantBalance, req.Signers)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) listReconciliations(c *gin.Context) {
	recs, err := h.svc.ListReconciliations(c.Request.Context(), c.Query("fund_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, recs)
}
