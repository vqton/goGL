package cash_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	_ "modernc.org/sqlite"

	appcash "goGL/internal/application/cash"
	domainaudit "goGL/internal/domain/audit"
	domaincash "goGL/internal/domain/cash"
	"goGL/internal/infrastructure/db"
	perscash "goGL/internal/infrastructure/persistence/cash"
)

func openSvcDB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	d, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d.Close() })

	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return d
}

type fakeAuditor struct {
	logs []*domainaudit.AuditLog
}

func (f *fakeAuditor) Record(_ context.Context, l *domainaudit.AuditLog) error {
	f.logs = append(f.logs, l)
	return nil
}

func newSvc(t *testing.T, d *sql.DB) (appcash.Service, *fakeAuditor) {
	t.Helper()
	a := &fakeAuditor{}
	return appcash.NewService(perscash.NewSqliteRepository(d), a), a
}

func validVoucher() *domaincash.Voucher {
	return &domaincash.Voucher{
		RefDate:          "2026-08-05",
		Type:             domaincash.VoucherReceive,
		Currency:         "VND",
		AmountMinor:      5_000_000,
		CounterpartyType: "customer",
		CounterpartyID:   "kh-1",
		CounterpartyName: "Công ty ABC",
		Description:      "Thu tiền bán hàng",
		Lines: []domaincash.VoucherLine{
			{Seq: 1, DebitAcc: "1111", AmountMinor: 5_000_000},
			{Seq: 2, CreditAcc: "131", AmountMinor: 3_000_000, ObjectID: "kh-1"},
			{Seq: 3, CreditAcc: "5111", AmountMinor: 2_000_000},
		},
	}
}

func mustCreateFund(t *testing.T, svc appcash.Service, id string) {
	t.Helper()
	if err := svc.CreateFund(context.Background(), &domaincash.Fund{
		ID: id, Name: "Quỹ VND", Currency: "VND", Account: "1111", IsActive: true,
	}); err != nil {
		t.Fatalf("create fund: %v", err)
	}
}

func TestService_CreateFundGeneratesIDAndLists(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()

	f := &domaincash.Fund{Name: "Quỹ VND", Currency: "VND", Account: "1111", IsActive: true}
	if err := svc.CreateFund(ctx, f); err != nil {
		t.Fatalf("create fund: %v", err)
	}
	if f.ID == "" {
		t.Fatal("expected service to assign fund ID")
	}

	list, err := svc.ListFunds(ctx)
	if err != nil {
		t.Fatalf("list funds: %v", err)
	}
	if len(list) != 1 || list[0].ID != f.ID {
		t.Fatalf("expected 1 fund, got %+v", list)
	}
}

func TestService_CreateVoucher_AssignsRefNoAndWords(t *testing.T) {
	svc, a := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create voucher: %v", err)
	}

	if v.ID == "" {
		t.Fatal("expected service to assign voucher ID")
	}
	if v.RefNo != "PT/2026-08/000001" {
		t.Fatalf("ref no = %q, want PT/2026-08/000001", v.RefNo)
	}
	if v.State != domaincash.VoucherDraft {
		t.Fatalf("state = %q, want draft", v.State)
	}
	if v.CreatedBy != "ketoan" {
		t.Fatalf("created_by = %q", v.CreatedBy)
	}
	if v.AmountWords != "năm triệu đồng" {
		t.Fatalf("amount words = %q", v.AmountWords)
	}

	last := a.logs[len(a.logs)-1]
	if last.Action != "voucher.create" || last.TargetID != v.ID || last.UserCode != "ketoan" {
		t.Fatalf("expected voucher.create audit log, got %+v", a.logs)
	}

	got, err := svc.GetVoucher(ctx, v.ID)
	if err != nil {
		t.Fatalf("get voucher: %v", err)
	}
	if got.RefNo != v.RefNo || got.AmountWords != "năm triệu đồng" {
		t.Fatalf("persisted voucher mismatch: %+v", got)
	}
}

func TestService_CreateVoucher_UnbalancedLines(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	v.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "1111", AmountMinor: 1_000_000},
		{Seq: 2, CreditAcc: "5111", AmountMinor: 2_000_000},
	}
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != domaincash.ErrInvalidLines {
		t.Fatalf("expected ErrInvalidLines, got %v", err)
	}
}

func TestService_CreateVoucher_MissingCashAccountLine(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	v.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "131", AmountMinor: 3_000_000},
		{Seq: 2, CreditAcc: "5111", AmountMinor: 2_000_000},
		{Seq: 3, CreditAcc: "138", AmountMinor: 1_000_000},
	}
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != domaincash.ErrInvalidLines {
		t.Fatalf("expected ErrInvalidLines (no cash line), got %v", err)
	}
}

func TestService_CreateVoucher_NoCounterparty(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	v.CounterpartyName = ""
	if err := svc.CreateVoucher(ctx, "ketoan", v); err == nil {
		t.Fatal("expected error for missing counterparty (R1)")
	}
}

func TestService_CreateVoucher_UnknownFund(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()

	v := validVoucher()
	v.FundID = "fund-nope"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != domaincash.ErrFundNotFound {
		t.Fatalf("expected ErrFundNotFound, got %v", err)
	}
}

func TestService_CreateVoucher_InactiveFund(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")
	if err := svc.CreateFund(ctx, &domaincash.Fund{ID: "fund-x", Name: "Quỹ đóng", Currency: "VND", Account: "1111"}); err != nil {
		t.Fatalf("create inactive fund: %v", err)
	}

	v := validVoucher()
	v.FundID = "fund-x"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != domaincash.ErrFundInactive {
		t.Fatalf("expected ErrFundInactive, got %v", err)
	}
}

func TestService_UpdateVoucher_DraftOnly(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}

	v.AmountMinor = 6_000_000
	v.Lines = []domaincash.VoucherLine{
		{Seq: 1, DebitAcc: "1111", AmountMinor: 6_000_000},
		{Seq: 2, CreditAcc: "131", AmountMinor: 4_000_000},
		{Seq: 3, CreditAcc: "5111", AmountMinor: 2_000_000},
	}
	if err := svc.UpdateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("update: %v", err)
	}
	if v.RefNo != "PT/2026-08/000001" || v.AmountWords != "sáu triệu đồng" {
		t.Fatalf("update must preserve refno and recompute words, got refno=%q words=%q", v.RefNo, v.AmountWords)
	}
	if v.CreatedBy != "ketoan" {
		t.Fatalf("update must preserve creator, got %q", v.CreatedBy)
	}

	got, err := svc.GetVoucher(ctx, v.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AmountMinor != 6_000_000 {
		t.Fatalf("updated amount not persisted: %+v", got)
	}
}

func TestService_UpdateVoucher_NotDraft(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ApproveVoucher(ctx, "giamdoc", v.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := svc.UpdateVoucher(ctx, "ketoan", v); err != domaincash.ErrWrongState {
		t.Fatalf("expected ErrWrongState, got %v", err)
	}
}

func TestService_ApproveVoucher(t *testing.T) {
	svc, a := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.ApproveVoucher(ctx, "giamdoc", v.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, err := svc.GetVoucher(ctx, v.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != domaincash.VoucherApproved || got.ApprovedBy != "giamdoc" || got.ApprovedAt == "" {
		t.Fatalf("approve not persisted: %+v", got)
	}
	if len(a.logs) == 0 || a.logs[len(a.logs)-1].Action != "voucher.approve" {
		t.Fatalf("expected voucher.approve audit log, got %+v", a.logs)
	}
}

func TestService_ApproveVoucher_SelfApproval(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ApproveVoucher(ctx, "ketoan", v.ID); err != domaincash.ErrSelfApproval {
		t.Fatalf("expected ErrSelfApproval, got %v", err)
	}
}

func TestService_ApproveVoucher_NotDraft(t *testing.T) {
	svc, _ := newSvc(t, openSvcDB(t))
	ctx := context.Background()
	mustCreateFund(t, svc, "fund-1")

	v := validVoucher()
	v.FundID = "fund-1"
	if err := svc.CreateVoucher(ctx, "ketoan", v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ApproveVoucher(ctx, "giamdoc", v.ID); err != nil {
		t.Fatalf("seed approved state: %v", err)
	}
	if err := svc.ApproveVoucher(ctx, "kttruong", v.ID); err != domaincash.ErrWrongState {
		t.Fatalf("expected ErrWrongState, got %v", err)
	}
}
