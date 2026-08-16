package ledger

import (
	"context"
	"database/sql"
	"errors"

	"goGL/internal/domain/ledger"
)

// DefaultChartSize returns the number of accounts in the default VAS chart.
func DefaultChartSize() int { return len(defaultChart) }

// defaultChart is a representative subset of the Vietnamese standard chart of
// accounts (Thông tư 99/2025/TT-BTC). Roots mirror the five ledger types;
// summary accounts never allow posting; leaves (AllowPost) take direct entries.
// Level-1..2 headings and level-3+ postable leaves keep the fixture small but
// true to the TT 99/2025 code numbering.
var defaultChart = []ledger.Account{
	// --- 1 Tài sản (assets) ---
	{Code: "1", Name: "Tài sản", Type: ledger.AccountAsset, Level: 1},
	{Code: "11", Name: "Tiền và các khoản tương đương tiền", Type: ledger.AccountAsset, ParentCode: "1", Level: 2},
	{Code: "111", Name: "Tiền", Type: ledger.AccountAsset, ParentCode: "11", Level: 3},
	{Code: "1111", Name: "Tiền mặt VND", Type: ledger.AccountAsset, ParentCode: "111", Level: 4, AllowPost: true},
	{Code: "1112", Name: "Tiền mặt ngoại tệ", Type: ledger.AccountAsset, ParentCode: "111", Level: 4, AllowPost: true},
	{Code: "112", Name: "Tiền gửi ngân hàng", Type: ledger.AccountAsset, ParentCode: "11", Level: 3},
	{Code: "1121", Name: "Tiền gửi ngân hàng VND", Type: ledger.AccountAsset, ParentCode: "112", Level: 4, AllowPost: true},
	{Code: "13", Name: "Các khoản phải thu ngắn hạn", Type: ledger.AccountAsset, ParentCode: "1", Level: 2},
	{Code: "131", Name: "Phải thu của khách hàng", Type: ledger.AccountAsset, ParentCode: "13", Level: 3},
	{Code: "1311", Name: "Phải thu bán hàng hóa", Type: ledger.AccountAsset, ParentCode: "131", Level: 4, AllowPost: true},
	{Code: "141", Name: "Tạm ứng", Type: ledger.AccountAsset, ParentCode: "1", Level: 2, AllowPost: true},
	{Code: "15", Name: "Hàng tồn kho", Type: ledger.AccountAsset, ParentCode: "1", Level: 2},
	{Code: "151", Name: "Nguyên vật liệu", Type: ledger.AccountAsset, ParentCode: "15", Level: 3},
	{Code: "1511", Name: "Nguyên vật liệu chính", Type: ledger.AccountAsset, ParentCode: "151", Level: 4, AllowPost: true},
	{Code: "21", Name: "Tài sản cố định hữu hình", Type: ledger.AccountAsset, ParentCode: "1", Level: 2},
	{Code: "211", Name: "Nguyên giá TSCĐ hữu hình", Type: ledger.AccountAsset, ParentCode: "21", Level: 3},
	{Code: "2111", Name: "Máy móc, thiết bị", Type: ledger.AccountAsset, ParentCode: "211", Level: 4, AllowPost: true},
	{Code: "214", Name: "Hao mòn lũy kế TSCĐ", Type: ledger.AccountAsset, ParentCode: "21", Level: 3},
	{Code: "2141", Name: "Hao mòn lũy kế TSCĐ hữu hình", Type: ledger.AccountAsset, ParentCode: "214", Level: 4, AllowPost: true},

	// --- 2 Nợ phải trả (liabilities) ---
	{Code: "2", Name: "Nợ phải trả", Type: ledger.AccountLiability, Level: 1},
	{Code: "31", Name: "Phải trả người bán", Type: ledger.AccountLiability, ParentCode: "2", Level: 2},
	{Code: "3111", Name: "Phải trả người bán trong nước", Type: ledger.AccountLiability, ParentCode: "31", Level: 3, AllowPost: true},
	{Code: "33", Name: "Thuế và các khoản phải nộp Nhà nước", Type: ledger.AccountLiability, ParentCode: "2", Level: 2},
	{Code: "333", Name: "Thuế GTGT phải nộp", Type: ledger.AccountLiability, ParentCode: "33", Level: 3},
	{Code: "3331", Name: "Thuế GTGT đầu ra", Type: ledger.AccountLiability, ParentCode: "333", Level: 4, AllowPost: true},
	{Code: "3334", Name: "Thuế thu nhập doanh nghiệp", Type: ledger.AccountLiability, ParentCode: "33", Level: 3, AllowPost: true},
	{Code: "34", Name: "Phải trả người lao động", Type: ledger.AccountLiability, ParentCode: "2", Level: 2},
	{Code: "341", Name: "Phải trả người lao động", Type: ledger.AccountLiability, ParentCode: "34", Level: 3},
	{Code: "3411", Name: "Phải trả lương", Type: ledger.AccountLiability, ParentCode: "341", Level: 4, AllowPost: true},
	{Code: "35", Name: "Vay và nợ thuê tài chính", Type: ledger.AccountLiability, ParentCode: "2", Level: 2},
	{Code: "3511", Name: "Vay ngắn hạn", Type: ledger.AccountLiability, ParentCode: "35", Level: 3, AllowPost: true},

	// --- 3 Vốn chủ sở hữu (equity) ---
	{Code: "3", Name: "Vốn chủ sở hữu", Type: ledger.AccountEquity, Level: 1},
	{Code: "41", Name: "Vốn chủ sở hữu", Type: ledger.AccountEquity, ParentCode: "3", Level: 2},
	{Code: "411", Name: "Vốn góp của chủ sở hữu", Type: ledger.AccountEquity, ParentCode: "41", Level: 3},
	{Code: "4111", Name: "Vốn đầu tư của chủ sở hữu", Type: ledger.AccountEquity, ParentCode: "411", Level: 4, AllowPost: true},
	{Code: "421", Name: "Lợi nhuận sau thuế chưa phân phối", Type: ledger.AccountEquity, ParentCode: "41", Level: 3},
	{Code: "4211", Name: "Lợi nhuận sau thuế chưa phân phối năm nay", Type: ledger.AccountEquity, ParentCode: "421", Level: 4, AllowPost: true},

	// --- 4 Doanh thu (revenue) ---
	{Code: "4", Name: "Doanh thu", Type: ledger.AccountRevenue, Level: 1},
	{Code: "51", Name: "Doanh thu bán hàng và cung cấp dịch vụ", Type: ledger.AccountRevenue, ParentCode: "4", Level: 2},
	{Code: "511", Name: "Doanh thu bán hàng hóa", Type: ledger.AccountRevenue, ParentCode: "51", Level: 3},
	{Code: "5111", Name: "Doanh thu bán hàng hóa", Type: ledger.AccountRevenue, ParentCode: "511", Level: 4, AllowPost: true},
	{Code: "5113", Name: "Doanh thu cung cấp dịch vụ", Type: ledger.AccountRevenue, ParentCode: "511", Level: 4, AllowPost: true},
	{Code: "515", Name: "Doanh thu hoạt động tài chính", Type: ledger.AccountRevenue, ParentCode: "51", Level: 3},
	{Code: "5151", Name: "Doanh thu tiền lãi", Type: ledger.AccountRevenue, ParentCode: "515", Level: 4, AllowPost: true},
	{Code: "71", Name: "Thu nhập khác", Type: ledger.AccountRevenue, ParentCode: "4", Level: 2},
	{Code: "711", Name: "Thu nhập khác", Type: ledger.AccountRevenue, ParentCode: "71", Level: 3},
	{Code: "7111", Name: "Thu nhập từ thanh lý tài sản", Type: ledger.AccountRevenue, ParentCode: "711", Level: 4, AllowPost: true},

	// --- 5 Chi phí (expenses) ---
	{Code: "5", Name: "Chi phí", Type: ledger.AccountExpense, Level: 1},
	{Code: "61", Name: "Giá vốn hàng bán", Type: ledger.AccountExpense, ParentCode: "5", Level: 2},
	{Code: "611", Name: "Giá vốn hàng bán", Type: ledger.AccountExpense, ParentCode: "61", Level: 3},
	{Code: "6111", Name: "Giá vốn hàng hóa", Type: ledger.AccountExpense, ParentCode: "611", Level: 4, AllowPost: true},
	{Code: "62", Name: "Chi phí bán hàng", Type: ledger.AccountExpense, ParentCode: "5", Level: 2},
	{Code: "621", Name: "Chi phí bán hàng", Type: ledger.AccountExpense, ParentCode: "62", Level: 3},
	{Code: "6211", Name: "Chi phí nhân viên bán hàng", Type: ledger.AccountExpense, ParentCode: "621", Level: 4, AllowPost: true},
	{Code: "63", Name: "Chi phí quản lý doanh nghiệp", Type: ledger.AccountExpense, ParentCode: "5", Level: 2},
	{Code: "631", Name: "Chi phí quản lý doanh nghiệp", Type: ledger.AccountExpense, ParentCode: "63", Level: 3},
	{Code: "6311", Name: "Chi phí lương nhân viên quản lý", Type: ledger.AccountExpense, ParentCode: "631", Level: 4, AllowPost: true},
	{Code: "6312", Name: "Chi phí khấu hao tài sản cố định", Type: ledger.AccountExpense, ParentCode: "631", Level: 4, AllowPost: true},
	{Code: "64", Name: "Chi phí hoạt động tài chính", Type: ledger.AccountExpense, ParentCode: "5", Level: 2},
	{Code: "641", Name: "Chi phí hoạt động tài chính", Type: ledger.AccountExpense, ParentCode: "64", Level: 3},
	{Code: "6411", Name: "Chi phí lãi vay", Type: ledger.AccountExpense, ParentCode: "641", Level: 4, AllowPost: true},
	{Code: "81", Name: "Chi phí khác", Type: ledger.AccountExpense, ParentCode: "5", Level: 2},
	{Code: "811", Name: "Chi phí khác", Type: ledger.AccountExpense, ParentCode: "81", Level: 3},
	{Code: "8111", Name: "Chi phí phạt, xử lý vi phạm", Type: ledger.AccountExpense, ParentCode: "811", Level: 4, AllowPost: true},
}

// SeedDefaultAccounts upserts the VAS chart onto a fresh store. It is
// idempotent: accounts that already exist are left untouched so runtime edits
// (renames, status changes) survive restarts. Returns the number of accounts
// created.
func SeedDefaultAccounts(ctx context.Context, repo ledger.Repository) (int, error) {
	created := 0
	for _, a := range defaultChart {
		existing, err := repo.GetAccountByCode(ctx, a.Code)
		if err == nil && existing != nil {
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return created, err
		}
		a.ID = ledger.RowID("account", a.Code)
		if a.Status == "" {
			a.Status = ledger.AccountActive
		}
		if err := repo.CreateAccount(ctx, &a); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
