package ledger

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	appledger "goGL/internal/application/ledger"
	domainledger "goGL/internal/domain/ledger"
)

type Handler struct {
	svc            appledger.Service
	identityHeader string
}

func NewHandler(svc appledger.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/ledger")
	g.GET("/accounts", h.listAccounts)
	g.POST("/accounts", h.createAccount)
	g.GET("/accounts/:id", h.getAccount)
	g.PATCH("/accounts/:id", h.updateAccount)
	g.GET("/periods", h.listPeriods)
	g.POST("/periods/:id/open", h.openPeriod)
	g.POST("/periods/:id/close", h.closePeriod)
	g.POST("/periods/:id/reopen", h.reopenPeriod)
	g.POST("/entries", h.createEntry)
	g.GET("/entries", h.listEntries)
	g.GET("/entries/:id", h.getEntry)
	g.POST("/entries/:id/post", h.postEntry)
	g.DELETE("/entries/:id", h.deleteEntry)
	g.GET("/books/general-journal", h.generalJournalBook)
	g.GET("/books/ledger", h.ledgerBook)
	g.GET("/books/detail", h.detailBook)
	g.GET("/books/trial-balance", h.trialBalanceBook)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

func respondError(c *gin.Context, err error) {
	var code int
	switch {
	case errors.Is(err, sql.ErrNoRows),
		errors.Is(err, domainledger.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, domainledger.ErrPeriodClosed),
		errors.Is(err, domainledger.ErrWrongState),
		errors.Is(err, domainledger.ErrDuplicateSource):
		code = http.StatusConflict
	case errors.Is(err, domainledger.ErrInvalidDate),
		errors.Is(err, domainledger.ErrUnbalanced),
		errors.Is(err, domainledger.ErrInvalidLine),
		errors.Is(err, domainledger.ErrAccountNotFound),
		errors.Is(err, domainledger.ErrAccountInactive),
		errors.Is(err, domainledger.ErrReversalMismatch),
		errors.Is(err, domainledger.ErrInvalidAccount),
		errors.Is(err, domainledger.ErrInvalidType),
		errors.Is(err, domainledger.ErrInvalidLevel),
		errors.Is(err, domainledger.ErrParentNotFound),
		errors.Is(err, domainledger.ErrInvalidHierarchy),
		errors.Is(err, domainledger.ErrTypeMismatch),
		errors.Is(err, domainledger.ErrInvalidPeriod),
		errors.Is(err, domainledger.ErrInvalidRange),
		errors.Is(err, domainledger.ErrCloseReasonRequired):
		code = http.StatusUnprocessableEntity
	default:
		code = http.StatusBadRequest
	}
	c.AbortWithStatusJSON(code, gin.H{"error": gin.H{"code": http.StatusText(code), "message": err.Error()}})
}

func (h *Handler) createEntry(c *gin.Context) {
	var e domainledger.JournalEntry
	if err := c.ShouldBindJSON(&e); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	created, err := h.svc.CreateEntry(c.Request.Context(), h.actor(c), &e)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getEntry(c *gin.Context) {
	e, err := h.svc.GetEntry(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, e)
}

// postEntry transitions a DRAFT entry to POSTED (spec 5.2). A re-post is
// idempotent: it returns the already-posted entry (R5).
func (h *Handler) postEntry(c *gin.Context) {
	e, err := h.svc.PostEntry(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, e)
}

// deleteEntry removes a DRAFT entry only (R7); POSTED entries are append-only.
func (h *Handler) deleteEntry(c *gin.Context) {
	if err := h.svc.DeleteEntry(c.Request.Context(), h.actor(c), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listEntries(c *gin.Context) {
	filter := domainledger.EntryFilter{
		Period:   c.Query("period"),
		Source:   domainledger.EntrySource(c.Query("source")),
		Status:   domainledger.EntryStatus(c.Query("status")),
		FromDate: c.Query("from"),
		ToDate:   c.Query("to"),
	}
	entries, err := h.svc.ListEntries(c.Request.Context(), filter)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *Handler) listAccounts(c *gin.Context) {
	filter := domainledger.AccountFilter{
		Type:       domainledger.AccountType(c.Query("type")),
		ParentCode: c.Query("parent"),
		Q:          c.Query("q"),
	}
	accounts, err := h.svc.ListAccounts(c.Request.Context(), filter)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *Handler) createAccount(c *gin.Context) {
	var a domainledger.Account
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateAccount(c.Request.Context(), h.actor(c), &a); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) getAccount(c *gin.Context) {
	a, err := h.svc.GetAccount(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) updateAccount(c *gin.Context) {
	var a domainledger.Account
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a.ID = c.Param("id")
	if err := h.svc.UpdateAccount(c.Request.Context(), h.actor(c), &a); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) listPeriods(c *gin.Context) {
	periods, err := h.svc.ListPeriods(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, periods)
}

func (h *Handler) openPeriod(c *gin.Context) {
	p, err := h.svc.OpenPeriod(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

type reasonRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) closePeriod(c *gin.Context) {
	var req reasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.svc.ClosePeriod(c.Request.Context(), h.actor(c), c.Param("id"), req.Reason)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) reopenPeriod(c *gin.Context) {
	var req reasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.svc.ReopenPeriod(c.Request.Context(), h.actor(c), c.Param("id"), req.Reason)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// --- P3.2 — statutory book endpoints. The four GETs under /books wrap the
// read-model with period/account filters and 1-based page/page_size paging.
// The service keeps the full row count in book.Total; Totals and balances are
// always computed over the whole book, never the page. ---

const (
	defaultPageSize = 100
	maxPageSize     = 1000
)

// bookPage parses the 1-based page/page_size query pair into an offset/limit
// window. Absent params mean "no paging"; page_size is capped at maxPageSize.
// A malformed value is a 400 BAD_REQUEST, like other handler-level parse
// errors.
func (h *Handler) bookPage(c *gin.Context) (*domainledger.Page, bool) {
	q := c.Query("page")
	ps := c.Query("page_size")
	if q == "" && ps == "" {
		return nil, true
	}
	page, size := 1, defaultPageSize
	var err error
	if q != "" {
		page, err = strconv.Atoi(q)
		if err != nil || page < 1 {
			badRequest(c, "page must be a positive integer")
			return nil, false
		}
	}
	if ps != "" {
		size, err = strconv.Atoi(ps)
		if err != nil || size < 1 {
			badRequest(c, "page_size must be a positive integer")
			return nil, false
		}
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return &domainledger.Page{Offset: (page - 1) * size, Limit: size}, true
}

func badRequest(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": message}})
}

func (h *Handler) generalJournalBook(c *gin.Context) {
	page, ok := h.bookPage(c)
	if !ok {
		return
	}
	book, err := h.svc.GetGeneralJournal(c.Request.Context(), c.Query("from"), c.Query("to"), page)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) ledgerBook(c *gin.Context) {
	page, ok := h.bookPage(c)
	if !ok {
		return
	}
	book, err := h.svc.GetLedgerBook(c.Request.Context(), c.Query("account"), c.Query("from"), c.Query("to"), page)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) detailBook(c *gin.Context) {
	page, ok := h.bookPage(c)
	if !ok {
		return
	}
	book, err := h.svc.GetDetailBook(c.Request.Context(), c.Query("account"), c.Query("from"), c.Query("to"), page)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) trialBalanceBook(c *gin.Context) {
	page, ok := h.bookPage(c)
	if !ok {
		return
	}
	tb, err := h.svc.GetTrialBalance(c.Request.Context(), c.Query("period"), page)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, tb)
}
