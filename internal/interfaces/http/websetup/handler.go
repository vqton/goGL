// Package websetup serves the server-rendered setup wizard / status
// dashboard under /setup/*. Mutations are HTML form posts that call the same
// application service as the JSON API; the actor is resolved from the identity
// header (dev seam). Page templates live in web/templates/setup/ and are
// parsed into the shared html/template set by the root web handler
// (web/handler.go), so this handler only registers routes. All template define
// names are prefixed "setup_*" because html/template names are global across
// the shared set.
package websetup

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appprint "goGL/internal/application/cash/print"
	appsetup "goGL/internal/application/setup"
	"goGL/internal/domain/core"
	domainsetup "goGL/internal/domain/setup"
)

type Handler struct {
	svc            appsetup.Service
	identityHeader string
}

func NewHandler(svc appsetup.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

// Funcs returns the template helpers the /setup pages rely on. They must be
// registered on the shared template set (web/handler.go) because html/template
// cannot invoke argument-carrying functions stored in the data map.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"money":       money,
		"fmtDate":     fmtDate,
		"fmtTs":       fmtTs,
		"sumDebit":    sumDebit,
		"sumCredit":   sumCredit,
		"statusLabel": statusLabel,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/setup")
	g.GET("", h.wizard)
	g.GET("/start", h.startForm)
	g.POST("/start", h.start)
	g.GET("/accounts", h.accounts)
	g.GET("/balances", h.balances)
	g.POST("/balances", h.saveBalance)
	g.POST("/balances/:id/delete", h.deleteBalance)
	g.POST("/balances/check", h.check)
	g.POST("/balances/lock", h.lock)
	g.POST("/balances/reopen", h.reopen)
	g.POST("/activate", h.activate)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

// requireActor fails closed when the identity header is absent. The web UI
// sits outside the /api/v1 Casbin middleware, so its mutating actions must
// refuse anonymous requests themselves (same pattern as webcash/webledger).
func (h *Handler) requireActor(c *gin.Context) bool {
	if h.actor(c) != "" {
		return true
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "missing identity header"}})
	return false
}

// wizard is the /setup dashboard: it shows the current status, the step
// checklist, the profile summary and the next action for the current status.
func (h *Handler) wizard(c *gin.Context) {
	ctx := c.Request.Context()
	view, err := h.svc.Status(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	// R13: recent setup audit trail for the dashboard (best effort — a read
	// failure must not take down the status page).
	trail, _ := h.svc.AuditTrail(ctx, "setup", 15)
	c.HTML(http.StatusOK, "setup_wizard", gin.H{
		"View":  view,
		"Trail": trail,
		"Err":   c.Query("err"),
		"Ok":    c.Query("ok"),
	})
}

// startForm renders step 1-2: company profile + accounting regime + fiscal
// year. Idempotent resume: pre-fills from the current profile when present.
func (h *Handler) startForm(c *gin.Context) {
	ctx := c.Request.Context()
	view, err := h.svc.Status(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "setup_start", gin.H{
		"View": view,
		"Err":  c.Query("err"),
	})
}

// start runs Initialize (steps 1-3: profile, regime, fiscal year, seed
// accounts, open periods — idempotent resume from the current status).
func (h *Handler) start(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		redirectErr(c, "/setup/start", "Biểu mẫu không hợp lệ")
		return
	}
	f := c.Request.PostForm
	profile := &domainsetup.CompanyProfile{
		ID:                  domainsetup.ProfileID,
		Name:                strings.TrimSpace(f.Get("name")),
		NameEN:              strings.TrimSpace(f.Get("name_en")),
		TaxCode:             domainsetup.NormalizeTaxCode(f.Get("tax_code")),
		BudgetUnitCode:      strings.TrimSpace(f.Get("budget_unit_code")),
		Address:             strings.TrimSpace(f.Get("address")),
		LegalRepresentative: strings.TrimSpace(f.Get("legal_representative")),
		CompanyType:         strings.TrimSpace(f.Get("company_type")),
		Industry:            strings.TrimSpace(f.Get("industry")),
		AccountingCurrency:  "VND",
		AccountingRegime:    f.Get("regime"),
		FiscalYearStart:     f.Get("fiscal_year_start"),
		BooksStartDate:      f.Get("books_start_date"),
	}

	view, err := h.svc.Initialize(c.Request.Context(), &appsetup.InitializeRequest{
		Profile:         profile,
		Regime:          f.Get("regime"),
		FiscalYearStart: f.Get("fiscal_year_start"),
		SeedAccounts:    f.Get("seed_accounts") == "on",
		OpenPeriods:     f.Get("open_periods") == "on",
	}, h.actor(c))
	if err != nil {
		redirectErr(c, "/setup/start", err.Error())
		return
	}
	_ = view
	c.Redirect(http.StatusSeeOther, "/setup?ok="+url.QueryEscape("Đã lưu. Bước tiếp theo: nhập số dư đầu kỳ."))
}

// accounts renders step 3: a read-only preview of the seeded COA (via the
// ledger seam) so the operator can confirm the chart before entering balances.
func (h *Handler) accounts(c *gin.Context) {
	ctx := c.Request.Context()
	view, err := h.svc.Status(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	preview, err := h.svc.PreviewAccounts(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "setup_accounts", gin.H{
		"View":    view,
		"Preview": preview,
	})
}

// balances renders step 4: the opening-balance table, the live balance check
// banner, the add-row form and the lock/reopen actions.
func (h *Handler) balances(c *gin.Context) {
	ctx := c.Request.Context()
	view, err := h.svc.Status(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	list, err := h.svc.ListBalances(ctx, "")
	if err != nil {
		h.fail(c, err)
		return
	}
	check, err := h.svc.CheckBalances(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "setup_balances", gin.H{
		"View":  view,
		"List":  list,
		"Check": check,
		"Err":   c.Query("err"),
		"Ok":    c.Query("ok"),
		"Today": time.Now().Format("2006-01-02"),
	})
}

// saveBalance adds/upserts one opening balance row (R7: deterministic id).
func (h *Handler) saveBalance(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		redirectErr(c, "/setup/balances", "Biểu mẫu không hợp lệ")
		return
	}
	f := c.Request.PostForm
	debit, err := parseMinor(f.Get("debit"))
	if err != nil {
		redirectErr(c, "/setup/balances", "Số tiền Nợ không hợp lệ")
		return
	}
	credit, err := parseMinor(f.Get("credit"))
	if err != nil {
		redirectErr(c, "/setup/balances", "Số tiền Có không hợp lệ")
		return
	}
	b := &domainsetup.OpeningBalance{
		AccountCode: strings.TrimSpace(f.Get("account_code")),
		ObjectType:  strings.TrimSpace(f.Get("object_type")),
		ObjectCode:  strings.TrimSpace(f.Get("object_code")),
		Period:      core.Period{},
		Debit:       core.Money{AmountMinor: debit, Currency: "VND"},
		Credit:      core.Money{AmountMinor: credit, Currency: "VND"},
		Status:      domainsetup.BalanceDraft,
	}
	if _, err := h.svc.SaveBalance(c.Request.Context(), b, h.actor(c)); err != nil {
		redirectErr(c, "/setup/balances", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/setup/balances?ok="+url.QueryEscape("Đã thêm số dư "+b.AccountCode))
}

func (h *Handler) deleteBalance(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := h.svc.DeleteBalance(c.Request.Context(), c.Param("id"), h.actor(c)); err != nil {
		redirectErr(c, "/setup/balances", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/setup/balances?ok="+url.QueryEscape("Đã xóa số dư"))
}

// check re-runs the R9 balance check and renders the balances page with the
// fresh banner (form posts only; the page already shows the live check).
func (h *Handler) check(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	_, err := h.svc.CheckBalances(c.Request.Context())
	if err != nil {
		redirectErr(c, "/setup/balances", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/setup/balances?ok="+url.QueryEscape("Đã kiểm tra cân đối"))
}

func (h *Handler) lock(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := h.svc.Lock(c.Request.Context(), h.actor(c)); err != nil {
		redirectErr(c, "/setup/balances", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/setup/balances?ok="+url.QueryEscape("Đã khóa số dư đầu kỳ"))
}

func (h *Handler) reopen(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		redirectErr(c, "/setup/balances", "Biểu mẫu không hợp lệ")
		return
	}
	reason := strings.TrimSpace(c.Request.PostForm.Get("reason"))
	if reason == "" {
		redirectErr(c, "/setup/balances", "Cần nêu lý do mở lại (bắt buộc — R12)")
		return
	}
	if err := h.svc.Reopen(c.Request.Context(), h.actor(c), reason); err != nil {
		redirectErr(c, "/setup/balances", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/setup/balances?ok="+url.QueryEscape("Đã mở lại số dư đầu kỳ"))
}

func (h *Handler) activate(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := h.svc.Activate(c.Request.Context(), h.actor(c)); err != nil {
		redirectErr(c, "/setup/balances", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/setup?ok="+url.QueryEscape("Đã kích hoạt hệ thống"))
}

// --- helpers ---------------------------------------------------------------

func redirectErr(c *gin.Context, to, msg string) {
	c.Redirect(http.StatusSeeOther, to+"?err="+url.QueryEscape(msg))
}

func parseMinor(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

func (h *Handler) fail(c *gin.Context, err error) {
	c.HTML(http.StatusInternalServerError, "setup_error", gin.H{"Err": err.Error()})
}

func money(n int64) string { return appprint.FormatVN(n) }

// fmtDate converts YYYY-MM-DD to dd/MM/yyyy for display.
func fmtDate(v string) string {
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return v
	}
	return t.Format("02/01/2006")
}

// fmtTs converts an RFC3339 audit timestamp to dd/MM/yyyy HH:mm for display.
func fmtTs(v string) string {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	return t.Format("02/01/2006 15:04")
}

func sumDebit(bs []*domainsetup.OpeningBalance) int64 {
	var n int64
	for _, b := range bs {
		n += b.Debit.AmountMinor
	}
	return n
}

func sumCredit(bs []*domainsetup.OpeningBalance) int64 {
	var n int64
	for _, b := range bs {
		n += b.Credit.AmountMinor
	}
	return n
}

func statusLabel(s domainsetup.SetupStatus) string { return s.Label() }
