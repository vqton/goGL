// Package webcash serves the server-rendered cashier UI under /cash/*.
// Mutations are HTML form posts that call the same application service as the
// JSON API; the actor is resolved from the identity header (dev seam). The
// page templates are loaded by the root web handler (web/templates/*.html +
// web/templates/cash/*.html share one html/template set), so this handler only
// registers routes.
package webcash

import (
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/cash"
	appprint "goGL/internal/application/cash/print"
	domaincash "goGL/internal/domain/cash"
)

type Handler struct {
	svc            cash.Service
	identityHeader string
}

func NewHandler(svc cash.Service, identityHeader string) *Handler {
	return &Handler{svc: svc, identityHeader: identityHeader}
}

// Funcs returns the template helpers the /cash pages rely on. They must be
// registered on the shared template set (web/handler.go) because html/template
// cannot invoke argument-carrying functions stored in the data map.
func Funcs() template.FuncMap {
	return template.FuncMap{
		"money":      money,
		"stateLabel": stateLabel,
		"stateBadge": stateBadge,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	c := r.Group("/cash")
	c.GET("", h.dashboard)
	c.GET("/vouchers", h.vouchers)
	c.GET("/vouchers/new", h.newVoucher)
	c.POST("/vouchers", h.createVoucher)
	c.GET("/vouchers/:id", h.voucherDetail)
	c.POST("/vouchers/:id/approve", h.approveVoucher)
	c.POST("/vouchers/:id/post", h.postVoucher)
	c.POST("/vouchers/:id/void", h.voidVoucher)
	c.GET("/book", h.book)
	c.GET("/close-day", h.closeDayForm)
	c.POST("/close-day", h.closeDay)
	c.GET("/reconcile", h.reconcileForm)
	c.POST("/reconcile", h.reconcile)
	c.GET("/print/voucher/:id", h.printVoucher)
	c.GET("/print/s07", h.printS07)
	c.GET("/export/vouchers.csv", h.exportVouchers)
	c.GET("/export/book.csv", h.exportBook)
	c.GET("/export/reconciliations.csv", h.exportReconciliations)
}

func (h *Handler) actor(c *gin.Context) string {
	return c.GetHeader(h.identityHeader)
}

// requireActor fails closed when the identity header is absent. The web UI
// sits outside the /api/v1 Casbin middleware, so its mutating actions must
// refuse anonymous requests themselves (T5.2).
func (h *Handler) requireActor(c *gin.Context) bool {
	if h.actor(c) != "" {
		return true
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "missing identity header"}})
	return false
}

func money(n int64) string { return appprint.FormatVN(n) }

func stateLabel(s domaincash.VoucherState) string {
	switch s {
	case domaincash.VoucherApproved:
		return "Đã duyệt"
	case domaincash.VoucherPosted:
		return "Đã ghi sổ"
	case domaincash.VoucherReconciled:
		return "Đã đối chiếu"
	case domaincash.VoucherVoided:
		return "Đã hủy"
	default:
		return "Nháp"
	}
}

func stateBadge(s domaincash.VoucherState) string {
	switch s {
	case domaincash.VoucherApproved:
		return "bg-blue-100 text-blue-700"
	case domaincash.VoucherPosted:
		return "bg-emerald-100 text-emerald-700"
	case domaincash.VoucherReconciled:
		return "bg-slate-200 text-slate-700"
	case domaincash.VoucherVoided:
		return "bg-red-100 text-red-700 line-through"
	default:
		return "bg-slate-100 text-slate-500"
	}
}

func (h *Handler) funds(c *gin.Context) ([]*domaincash.Fund, error) {
	return h.svc.ListFunds(c.Request.Context())
}

func (h *Handler) fundByID(ctx context.Context, id string) (*domaincash.Fund, error) {
	funds, err := h.svc.ListFunds(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range funds {
		if f.ID == id {
			return f, nil
		}
	}
	return nil, domaincash.ErrFundNotFound
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
	c.HTML(http.StatusInternalServerError, "error", gin.H{"Message": err.Error()})
}

func (h *Handler) dashboard(c *gin.Context) {
	ctx := c.Request.Context()
	funds, err := h.svc.ListFunds(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	counts, err := h.svc.ListCashCounts(ctx, "")
	if err != nil {
		h.fail(c, err)
		return
	}
	recs, err := h.svc.ListReconciliations(ctx, "")
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "dashboard", gin.H{
		"Funds":           funds,
		"Counts":          counts,
		"Reconciliations": recs,
		"Money":           money,
		"Now":             time.Now().Format("02/01/2006"),
		"Actor":           h.actor(c),
	})
}

func (h *Handler) vouchers(c *gin.Context) {
	ctx := c.Request.Context()
	funds, err := h.svc.ListFunds(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	filter := domaincash.VoucherFilter{
		FundID: c.Query("fund_id"),
		State:  domaincash.VoucherState(c.Query("state")),
		Type:   domaincash.VoucherType(c.Query("type")),
	}
	list, err := h.svc.ListVouchers(ctx, filter)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "vouchers", gin.H{
		"Vouchers":   list,
		"Funds":      funds,
		"Filter":     filter,
		"Money":      money,
		"StateLabel": stateLabel,
		"StateBadge": stateBadge,
		"Flash":      flash(c),
		"Actor":      h.actor(c),
	})
}

func (h *Handler) newVoucher(c *gin.Context) {
	funds, err := h.funds(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "voucher_form", gin.H{
		"Funds": funds,
		"Today": time.Now().Format("2006-01-02"),
		"Flash": flash(c),
		"Actor": h.actor(c),
	})
}

func (h *Handler) createVoucher(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		h.fail(c, err)
		return
	}
	f := c.Request.PostForm
	amount, err := strconv.ParseInt(f.Get("amount_minor"), 10, 64)
	if err != nil {
		redirectErr(c, "/cash/vouchers/new", "Số tiền không hợp lệ")
		return
	}
	fund, err := h.fundByID(c.Request.Context(), f.Get("fund_id"))
	if err != nil {
		redirectErr(c, "/cash/vouchers/new", "Quỹ không hợp lệ")
		return
	}

	v := &domaincash.Voucher{
		Type:             domaincash.VoucherType(f.Get("type")),
		FundID:           fund.ID,
		RefDate:          f.Get("ref_date"),
		Currency:         f.Get("currency"),
		CounterpartyType: f.Get("counterparty_type"),
		CounterpartyID:   f.Get("counterparty_id"),
		CounterpartyName: f.Get("counterparty_name"),
		Description:      f.Get("description"),
		ReceiverName:     f.Get("receiver_name"),
		AmountMinor:      amount,
	}
	if v.RefDate == "" {
		v.RefDate = time.Now().Format("2006-01-02")
	}
	if fx := f.Get("fx_rate"); fx != "" {
		if x, err := strconv.ParseFloat(fx, 64); err == nil {
			v.FXRate = x
		}
	}

	other := f.Get("other_account")
	switch v.Type {
	case domaincash.VoucherReceive:
		v.Lines = []domaincash.VoucherLine{
			{Seq: 1, DebitAcc: fund.Account, AmountMinor: amount},
			{Seq: 2, CreditAcc: other, AmountMinor: amount},
		}
	default:
		v.Type = domaincash.VoucherPay
		v.Lines = []domaincash.VoucherLine{
			{Seq: 1, CreditAcc: fund.Account, AmountMinor: amount},
			{Seq: 2, DebitAcc: other, AmountMinor: amount},
		}
	}

	if err := h.svc.CreateVoucher(c.Request.Context(), h.actor(c), v); err != nil {
		redirectErr(c, "/cash/vouchers/new", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/cash/vouchers/"+v.ID+"?ok=Đã tạo phiếu "+v.RefNo)
}

func (h *Handler) voucherDetail(c *gin.Context) {
	ctx := c.Request.Context()
	v, err := h.svc.GetVoucher(ctx, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	fund, err := h.fundByID(ctx, v.FundID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "voucher_detail", gin.H{
		"V":          v,
		"Fund":       fund,
		"Money":      money,
		"StateLabel": stateLabel,
		"StateBadge": stateBadge,
		"Flash":      flash(c),
		"Actor":      h.actor(c),
	})
}

func (h *Handler) approveVoucher(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	id := c.Param("id")
	if _, err := h.svc.ApproveVoucher(c.Request.Context(), h.actor(c), id); err != nil {
		redirectErr(c, "/cash/vouchers/"+id, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/cash/vouchers/"+id+"?ok=Đã duyệt")
}

func (h *Handler) postVoucher(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	id := c.Param("id")
	if _, err := h.svc.PostVoucher(c.Request.Context(), h.actor(c), id); err != nil {
		redirectErr(c, "/cash/vouchers/"+id, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/cash/vouchers/"+id+"?ok=Đã ghi sổ")
}

func (h *Handler) voidVoucher(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	id := c.Param("id")
	if err := c.Request.ParseForm(); err != nil {
		h.fail(c, err)
		return
	}
	if _, err := h.svc.VoidVoucher(c.Request.Context(), h.actor(c), id, c.Request.PostForm.Get("reason")); err != nil {
		redirectErr(c, "/cash/vouchers/"+id, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/cash/vouchers/"+id+"?ok=Đã hủy")
}

func (h *Handler) book(c *gin.Context) {
	ctx := c.Request.Context()
	fundID, from, to := c.Query("fund_id"), c.Query("from"), c.Query("to")
	entries, err := h.svc.GetCashBook(ctx, fundID, from, to)
	if err != nil {
		h.fail(c, err)
		return
	}
	funds, err := h.svc.ListFunds(ctx)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "book", gin.H{
		"Entries": entries,
		"Funds":   funds,
		"FundID":  fundID,
		"From":    from,
		"To":      to,
		"Money":   money,
		"Actor":   h.actor(c),
	})
}

func (h *Handler) closeDayForm(c *gin.Context) {
	funds, err := h.funds(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "close_day", gin.H{
		"Funds": funds,
		"Today": time.Now().Format("2006-01-02"),
		"Flash": flash(c),
		"Actor": h.actor(c),
	})
}

func (h *Handler) closeDay(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		h.fail(c, err)
		return
	}
	f := c.Request.PostForm
	amount, err := strconv.ParseInt(f.Get("counted_amount"), 10, 64)
	if err != nil {
		redirectErr(c, "/cash/close-day", "Số tiền kiểm kê không hợp lệ")
		return
	}
	var participants []string
	for _, p := range strings.Split(f.Get("participants"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			participants = append(participants, p)
		}
	}
	if _, err := h.svc.CloseDay(c.Request.Context(), h.actor(c), f.Get("fund_id"), f.Get("count_date"), amount, participants); err != nil {
		redirectErr(c, "/cash/close-day", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/cash?ok=Đã khóa sổ ngày")
}

func (h *Handler) reconcileForm(c *gin.Context) {
	funds, err := h.funds(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.HTML(http.StatusOK, "reconcile", gin.H{
		"Funds": funds,
		"Month": time.Now().Format("2006-01"),
		"Flash": flash(c),
		"Actor": h.actor(c),
	})
}

func (h *Handler) reconcile(c *gin.Context) {
	if !h.requireActor(c) {
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		h.fail(c, err)
		return
	}
	f := c.Request.PostForm
	balance, err := strconv.ParseInt(f.Get("accountant_balance"), 10, 64)
	if err != nil {
		redirectErr(c, "/cash/reconcile", "Số dư kế toán không hợp lệ")
		return
	}
	if _, err := h.svc.ReconcileMonth(c.Request.Context(), h.actor(c), f.Get("fund_id"), f.Get("period"), balance); err != nil {
		redirectErr(c, "/cash/reconcile", err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/cash?ok=Đã đối chiếu tháng")
}

func (h *Handler) printVoucher(c *gin.Context) {
	ctx := c.Request.Context()
	v, err := h.svc.GetVoucher(ctx, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	fund, err := h.fundByID(ctx, v.FundID)
	if err != nil {
		h.fail(c, err)
		return
	}
	out, err := appprint.VoucherForm(v, fund, "01")
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(out))
}

func (h *Handler) printS07(c *gin.Context) {
	ctx := c.Request.Context()
	fundID, from, to := c.Query("fund_id"), c.Query("from"), c.Query("to")
	fund, err := h.fundByID(ctx, fundID)
	if err != nil {
		h.fail(c, err)
		return
	}
	entries, err := h.svc.GetCashBook(ctx, fundID, from, to)
	if err != nil {
		h.fail(c, err)
		return
	}
	opening := int64(0)
	year := "2026"
	if len(entries) > 0 {
		opening = entries[0].Balance - entries[0].Receive - entries[0].Pay
		if len(entries[0].EntryDate) >= 4 {
			year = entries[0].EntryDate[:4]
		}
	}
	if len(from) >= 4 {
		year = from[:4]
	}
	out, err := appprint.S07DN(appprint.S07Data{
		Fund:    fund,
		Entries: entries,
		Opening: opening,
		Year:    year,
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(out))
}

// csvResponse streams a UTF-8 (BOM-prefixed) CSV so Excel opens Vietnamese
// columns correctly. filename is used in the Content-Disposition header.
func csvResponse(c *gin.Context, filename string, header []string, rows [][]string) {
	var b strings.Builder
	b.WriteString("\ufeff")
	w := csv.NewWriter(&b)
	_ = w.Write(header)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(b.String()))
}

func (h *Handler) exportVouchers(c *gin.Context) {
	filter := domaincash.VoucherFilter{
		FundID: c.Query("fund_id"),
		State:  domaincash.VoucherState(c.Query("state")),
		Type:   domaincash.VoucherType(c.Query("type")),
		From:   c.Query("from"),
		To:     c.Query("to"),
	}
	list, err := h.svc.ListVouchers(c.Request.Context(), filter)
	if err != nil {
		h.fail(c, err)
		return
	}
	header := []string{"Số phiếu", "Ngày", "Loại", "Đối tượng", "Diễn giải", "Số tiền", "Trạng thái"}
	rows := make([][]string, 0, len(list))
	for _, v := range list {
		loai := "Thu"
		if v.Type == domaincash.VoucherPay {
			loai = "Chi"
		}
		rows = append(rows, []string{
			v.RefNo, v.RefDate, loai, v.CounterpartyName, v.Description,
			strconv.FormatInt(v.AmountMinor, 10), stateLabel(v.State),
		})
	}
	csvResponse(c, "vouchers.csv", header, rows)
}

func (h *Handler) exportBook(c *gin.Context) {
	ctx := c.Request.Context()
	fundID, from, to := c.Query("fund_id"), c.Query("from"), c.Query("to")
	entries, err := h.svc.GetCashBook(ctx, fundID, from, to)
	if err != nil {
		h.fail(c, err)
		return
	}
	header := []string{"Ngày CT", "Số hiệu", "Diễn giải", "Thu", "Chi", "Tồn"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			e.VoucherDate, e.RefNo, e.Description,
			strconv.FormatInt(e.Receive, 10),
			strconv.FormatInt(e.Pay, 10),
			strconv.FormatInt(e.Balance, 10),
		})
	}
	csvResponse(c, "so-quy.csv", header, rows)
}

func (h *Handler) exportReconciliations(c *gin.Context) {
	recs, err := h.svc.ListReconciliations(c.Request.Context(), c.Query("fund_id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	header := []string{"Quỹ", "Kỳ", "Thủ quỹ", "Kế toán", "Chênh lệch", "Trạng thái", "Ngày lập"}
	rows := make([][]string, 0, len(recs))
	for _, r := range recs {
		state := "Chưa khớp"
		if r.State == "resolved" {
			state = "Đã khớp"
		}
		rows = append(rows, []string{
			r.FundID, r.Period,
			strconv.FormatInt(r.CashierBalance, 10),
			strconv.FormatInt(r.AccountantBalance, 10),
			strconv.FormatInt(r.Difference, 10), state, r.CreatedAt,
		})
	}
	csvResponse(c, "doi-chieu.csv", header, rows)
}
