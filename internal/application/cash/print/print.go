// Package print renders statutory VN cash forms (TT200/TT99) as printable HTML:
// phiếu thu/chi (01-TT, 02-TT), sổ quỹ tiền mặt (S07-DN, S07a-DN), biên bản
// kiểm kê quỹ and biên bản đối chiếu quỹ cuối tháng.
package print

import (
	"embed"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"goGL/internal/domain/cash"
)

//go:embed templates/*.html
var templatesFS embed.FS

var tpl *template.Template

func init() {
	fns := template.FuncMap{
		"fmtMoney": FormatVN,
		"fmtDate":  FormatDate,
		"words":    cash.AmountInWords,
		"join":     strings.Join,
	}
	tpl = template.Must(template.New("").Funcs(fns).ParseFS(templatesFS, "templates/*.html"))
}

// FormatVN renders an int64 minor-unit amount with Vietnamese thousands
// separators, e.g. 1234567 -> "1.234.567".
func FormatVN(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// FormatDate renders yyyy-mm-dd as dd/mm/yyyy. Unknown layouts pass through.
func FormatDate(d string) string {
	p := strings.Split(d, "-")
	if len(p) == 3 && len(p[0]) == 4 {
		return p[2] + "/" + p[1] + "/" + p[0]
	}
	return d
}

// VoucherForm renders a phiếu thu (Mẫu 01-TT) or phiếu chi (Mẫu 02-TT),
// selected by v.Type. bookNo is the quyển số.
func VoucherForm(v *cash.Voucher, fund *cash.Fund, bookNo string) (string, error) {
	title, formCode, partyLabel, signerLabel := "PHIẾU THU", "(Mẫu số: 01-TT)", "Họ tên người nộp tiền", "Người nộp tiền"
	if v.Type == cash.VoucherPay {
		title, formCode, partyLabel, signerLabel = "PHIẾU CHI", "(Mẫu số: 02-TT)", "Họ tên người nhận tiền", "Người nhận tiền"
	}
	debit, credit := formAccounts(v, fund)
	fx := ""
	if v.FXRate > 0 {
		fx = fmt.Sprintf("%.4f", v.FXRate)
	}
	data := struct {
		Title, FormCode, BookNo string
		Voucher                 *cash.Voucher
		Fund                    *cash.Fund
		DebitAcc, CreditAcc     string
		PartyLabel, SignerLabel string
		Party                   string
		FX                      string
	}{
		Title: title, FormCode: formCode, BookNo: bookNo,
		Voucher: v, Fund: fund,
		DebitAcc: debit, CreditAcc: credit,
		PartyLabel: partyLabel, SignerLabel: signerLabel,
		Party: v.CounterpartyName, FX: fx,
	}
	return render("voucher_form", data)
}

func formAccounts(v *cash.Voucher, fund *cash.Fund) (debit, credit string) {
	seen := map[string]bool{}
	var others []string
	for _, l := range v.Lines {
		for _, a := range []string{l.DebitAcc, l.CreditAcc} {
			if a == "" || strings.HasPrefix(a, fund.Account) || seen[a] {
				continue
			}
			seen[a] = true
			others = append(others, a)
		}
	}
	other := strings.Join(others, ", ")
	if v.Type == cash.VoucherReceive {
		return fund.Account, other
	}
	return other, fund.Account
}

// S07Data carries the cash book for S07-DN / S07a-DN rendering.
type S07Data struct {
	Fund    *cash.Fund
	Entries []*cash.CashBookEntry
	Opening int64
	Year    string
}

// S07DN renders the cashier's sổ quỹ tiền mặt (Mẫu S07-DN).
func S07DN(d S07Data) (string, error) {
	return renderS07(d, "SỔ QUỸ TIỀN MẶT", "S07-DN", "")
}

// S07aDN renders the accountant's parallel sổ quỹ tiền mặt (Mẫu S07a-DN).
func S07aDN(d S07Data) (string, error) {
	return renderS07(d, "SỔ QUỸ TIỀN MẶT", "S07a-DN", "Sổ kế toán ghi song song.")
}

func renderS07(d S07Data, title, formCode, note string) (string, error) {
	data := struct {
		Title, FormCode, Note string
		Fund                  *cash.Fund
		Entries               []*cash.CashBookEntry
		Opening               int64
		Year                  string
	}{
		Title: title, FormCode: formCode, Note: note,
		Fund: d.Fund, Entries: d.Entries, Opening: d.Opening, Year: d.Year,
	}
	return render("s07", data)
}

// KiemKeQuy renders the biên bản kiểm kê quỹ for a cash count.
func KiemKeQuy(fund *cash.Fund, c *cash.CashCount) (string, error) {
	return render("kiem_ke", struct {
		Fund  *cash.Fund
		Count *cash.CashCount
	}{fund, c})
}

// BienBanDoiChieu renders the biên bản đối chiếu quỹ cuối tháng for a
// monthly reconciliation (UC-5).
func BienBanDoiChieu(fund *cash.Fund, rec *cash.Reconciliation) (string, error) {
	month := rec.Period
	if p := strings.Split(rec.Period, "-"); len(p) == 2 {
		month = p[1] + "/" + p[0]
	}
	return render("doi_chieu", struct {
		Fund  *cash.Fund
		Rec   *cash.Reconciliation
		Month string
	}{fund, rec, month})
}

func render(name string, data any) (string, error) {
	var sb strings.Builder
	if err := tpl.ExecuteTemplate(&sb, name, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}
