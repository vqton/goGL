// Package webledger serves the server-rendered ledger UI under /ledger/*.
// Mutations are HTML form posts that call the same application service as the
// JSON API; the actor is resolved from the identity header (dev seam). The
// page templates are loaded by the root web handler (web/templates/*.html +
// web/templates/ledger/*.html share one html/template set), so this handler
// only registers routes. All template define names are prefixed "ledger_*"
// because html/template templates are global across the shared set.
package webledger

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appprint "goGL/internal/application/cash/print"
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

// Funcs returns the template helpers the /ledger pages rely on. They must be
// registered on the shared template set (web/handler.go) because html/template
// cannot invoke argument-carrying functions stored in the data map.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"money":       money,
		"statusLabel": statusLabel,
		"statusBadge": statusBadge,
		"sumDebit":    sumDebit,
		"sumCredit":   sumCredit,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	g := r.Group("/ledger")
	g.GET("", h.entries)
	g.GET("/entries", h.entries)
	g.GET("/entries/new", h.newEntry)
	g.POST("/entries", h.createEntry)
	g.GET("/entries/:id", h.entryDetail)
	g.POST("/entries/:id/post", h.postEntry)
	g.POST("/entries/:id/delete", h.deleteEntry)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

// requireActor fails closed when the identity header is absent. The web UI
// sits outside the /api/v1 Casbin middleware, so its mutating actions must
// refuse anonymous requests themselves (same pattern as webcash).
func (h *Handler) requireActor(c *gin.Context) bool {
	if h.actor(c) != "" {
		return true
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "missing identity header"}})
	return false
}

func money(n int64) string { return appprint.FormatVN(n) }

func statusLabel(s domainledger.EntryStatus) string {
	switch s {
	case domainledger.EntryPosted:
		return "Đã ghi sổ"
	case domainledger.EntryReversed:
		return "Đã điều chỉnh"
	default:
		return "Nháp"
	}
}

func statusBadge(s domainledger.EntryStatus) string {
	switch s {
	case domainledger.EntryPosted:
		return "bg-emerald-100 text-emerald-700"
	case domainledger.EntryReversed:
		return "bg-red-100 text-red-700"
	default:
		return "bg-slate-100 text-slate-500"
	}
}

func sumDebit(lines []domainledger.JournalLine) int64 {
	var total int64
	for _, l := range lines {
		total += l.Debit
	}
	return total
}

func sumCredit(lines []domainledger.JournalLine) int64 {
	var total int64
	for _, l := range lines {
		total += l.Credit
	}
	return total
}

func flash(c *gin.Context) gin.H {
	f := gin.H{}
	if e := c.Query("err"); e != "" {
		f["Err"] = e
	}
	if o := c.Query("ok"); o != "" {
		f["Ok"] = o
	}
	return f
}

func redirectErr(c *gin.Context, to, msg string) {
	c.Redirect(http.StatusSeeOther, to+"?err="+url.QueryEscape(msg))
}

func (h *Handler) fail(c *gin.Context, err error) {
	c.HTML(http.StatusInternalServerError, "ledger_error", gin.H{"Message": err.Error()})
}

func (h *Handler) entries(c *gin.Context) {
	ctx := c.Request.Context()
	filter := domainledger.EntryFilter{
		Period: c.Query("period"),
		Status: domainledger.EntryStatus(c.Query("status")),
		Source: domainledger.EntrySource(c.Query("source")),
	}
	list, err := h.svc.ListEntries(ctx, filter)
	if err != nil {
		h.fail(c, err)
		return
	}
	periods, err := h.svc.ListPeriods(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "ledger_entries", gin.H{
		"Entries":     list,
		"Periods":     periods,
		"Filter":      filter,
		"Money":       money,
		"StatusLabel": statusLabel,
		"StatusBadge": statusBadge,
		"SumDebit":    sumDebit,
		"SumCredit":   sumCredit,
		"Flash":       flash(c),
		"Actor":       h.actor(c),
	})
}

// postableAccounts returns the leaf accounts a manual entry may debit/credit
// (R3) — used for the account datalist in the entry form.
func (h *Handler) postableAccounts(ctx context.Context) ([]*domainledger.Account, error) {
	accounts, err := h.svc.ListAccounts(ctx, domainledger.AccountFilter{})
	if err != nil {
		return nil, err
	}
	out := []*domainledger.Account{}
	for _, a := range accounts {
		if a.AllowPost {
			out = append(out, a)
		}
	}
	return out, nil
}

func (h *Handler) newEntry(c *gin.Context) {
	accounts, err := h.postableAccounts(c.Request.Context())
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "ledger_entry_form", gin.H{
		"Accounts": accounts,
		"Today":    time.Now().Format("2006-01-02"),
		"Flash":    flash(c),
		"Actor":    h.actor(c),
	})
}

func at(vals []string, i int) string {
	if i >= 0 && i < len(vals) {
		return vals[i]
	}
	return ""
}

// parseLines rebuilds []JournalLine from the repeated form fields
// account_code/debit/credit/note. Rows with an empty account code are skipped.
func parseLines(f url.Values) []domainledger.JournalLine {
	codes := f["account_code"]
	out := []domainledger.JournalLine{}
	for i := range codes {
		code := strings.TrimSpace(codes[i])
		if code == "" {
			continue
		}
		line := domainledger.JournalLine{
			LineNo:      len(out) + 1,
			AccountCode: code,
			Note:        strings.TrimSpace(at(f["note"], i)),
		}
		if v, err := strconv.ParseInt(strings.TrimSpace(at(f["debit"], i)), 10, 64); err == nil {
			line.Debit = v
		}
		if v, err := strconv.ParseInt(strings.TrimSpace(at(f["credit"], i)), 10, 64); err == nil {
			line.Credit = v
		}
		out = append(out, line)
	}
	return out
}

// createEntry saves a manual journal entry. action=save keeps it as a draft;
// action=post creates the draft and immediately posts it (spec 5.2).
func (h *Handler) createEntry(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		h.fail(c, err)
		return
	}
	f := c.Request.PostForm
	lines := parseLines(f)
	if len(lines) == 0 {
		redirectErr(c, "/ledger/entries/new", "Chưa có dòng hạch toán nào")
		return
	}
	date := f.Get("voucher_date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	e := &domainledger.JournalEntry{
		VoucherDate: date,
		Description: f.Get("description"),
		Lines:       lines,
	}
	created, err := h.svc.CreateEntry(c.Request.Context(), h.actor(c), e)
	if err != nil {
		redirectErr(c, "/ledger/entries/new", err.Error())
		return
	}
	if f.Get("action") == "post" {
		posted, err := h.svc.PostEntry(c.Request.Context(), h.actor(c), created.ID)
		if err != nil {
			redirectErr(c, "/ledger/entries/"+created.ID, err.Error())
			return
		}
		c.Redirect(http.StatusSeeOther, "/ledger/entries/"+created.ID+"?ok=Đã ghi sổ "+posted.VoucherNo)
		return
	}
	c.Redirect(http.StatusSeeOther, "/ledger/entries/"+created.ID+"?ok=Đã lưu nháp")
}

func (h *Handler) entryDetail(c *gin.Context) {
	ctx := c.Request.Context()
	e, err := h.svc.GetEntry(ctx, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "ledger_entry_detail", gin.H{
		"Entry":       e,
		"Money":       money,
		"StatusLabel": statusLabel,
		"StatusBadge": statusBadge,
		"SumDebit":    sumDebit,
		"SumCredit":   sumCredit,
		"Flash":       flash(c),
		"Actor":       h.actor(c),
	})
}

func (h *Handler) postEntry(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	id := c.Param("id")
	posted, err := h.svc.PostEntry(c.Request.Context(), h.actor(c), id)
	if err != nil {
		redirectErr(c, "/ledger/entries/"+id, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/ledger/entries/"+id+"?ok=Đã ghi sổ "+posted.VoucherNo)
}

func (h *Handler) deleteEntry(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	id := c.Param("id")
	if err := h.svc.DeleteEntry(c.Request.Context(), h.actor(c), id); err != nil {
		redirectErr(c, "/ledger/entries/"+id, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/ledger/entries?ok=Đã xóa bút toán nháp")
}
