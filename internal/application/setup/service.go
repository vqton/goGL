package setup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/core"
	"goGL/internal/domain/ledger"
	"goGL/internal/domain/masterdata"
	"goGL/internal/domain/setup"
)

// Service is the setup application service — the initialize ORCHESTRATOR
// (docs/setup/02-spec). It owns the company profile + opening balances + the
// status machine, and orchestrates masterdata/ledger through narrow seams.
type Service interface {
	Status(ctx context.Context) (*StatusView, error)
	GetProfile(ctx context.Context) (*setup.CompanyProfile, error)
	Initialize(ctx context.Context, req *InitializeRequest, actor string) (*StatusView, error)
	UpdateProfile(ctx context.Context, p *setup.CompanyProfile, actor string) (*setup.CompanyProfile, error)

	SaveBalance(ctx context.Context, b *setup.OpeningBalance, actor string) (*setup.OpeningBalance, error)
	ListBalances(ctx context.Context, accountCode string) (*BalanceList, error)
	DeleteBalance(ctx context.Context, id, actor string) error
	CheckBalances(ctx context.Context) (*setup.BalanceCheck, error)
	PreviewAccounts(ctx context.Context) (*setup.AccountPreview, error)
	AuditTrail(ctx context.Context, module string, limit int) ([]*audit.AuditLog, error)
	ImportBalances(ctx context.Context, rows [][]string, actor string, dryRun bool) (*setup.ImportResult, error)
	GetImportReport(ctx context.Context, jobID string) (*setup.ImportJob, error)

	Lock(ctx context.Context, actor string) error
	Reopen(ctx context.Context, actor, reason string) error
	Activate(ctx context.Context, actor string) error
}

// StatusView is the GET /setup/status payload: current status, step checklist,
// the profile (when it exists) and the running balance-check summary.
type StatusView struct {
	Status   setup.SetupStatus       `json:"status"`
	Steps    []StatusStep            `json:"steps"`
	Profile  *setup.CompanyProfile   `json:"profile,omitempty"`
	Regime   string                  `json:"regime"`
	Check    *setup.BalanceCheck     `json:"balance_check,omitempty"`
	Balances []*setup.OpeningBalance `json:"balances,omitempty"`
}

// StatusStep is one checklist row on the wizard dashboard.
type StatusStep struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Done  bool   `json:"done"`
}

// BalanceList is the GET /setup/opening-balances payload.
type BalanceList struct {
	Balances []*setup.OpeningBalance `json:"balances"`
	Check    *setup.BalanceCheck     `json:"check"`
}

// InitializeRequest is the POST /setup/initialize body (docs/setup §5.1).
// Idempotent: it resumes from the current SetupStatus, re-checking each step
// before re-applying it, and never moves status backwards (R6).
type InitializeRequest struct {
	Profile         *setup.CompanyProfile `json:"profile"`
	Regime          string                `json:"regime"`
	FiscalYearStart string                `json:"fiscal_year_start"`
	SeedAccounts    bool                  `json:"seed_accounts"`
	OpenPeriods     bool                  `json:"open_periods"`
}

// Dependencies are the cross-module seams (Go interfaces, NOT HTTP —
// docs/setup §10). The concrete masterdata.Service / ledger.Service /
// audit.Service satisfy them structurally.
type Dependencies struct {
	Regime   RegimeManager
	Seeder   AccountSeeder
	Objects  ObjectLookup
	Periods  PeriodOpener
	Accounts AccountLookup
	Postings PostingLister
	Audit    Auditor
	Now      func() time.Time
}

// Seam interfaces — narrow, consumer-shaped slices of the provider modules.

// RegimeManager exposes the masterdata regime switch/read.
type RegimeManager interface {
	SetRegime(ctx context.Context, regime, actor string) error
	GetRegime(ctx context.Context) (string, error)
}

// AccountSeeder exposes the idempotent COA seed.
type AccountSeeder interface {
	SeedAccounts(ctx context.Context, actor string) (int, error)
}

// ObjectLookup resolves a đối tượng code into a masterdata record (R10).
type ObjectLookup interface {
	Get(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error)
}

// PeriodOpener opens the fiscal-year ledger periods.
type PeriodOpener interface {
	OpenPeriod(ctx context.Context, actor, id string) (*ledger.AccountingPeriod, error)
	ListPeriods(ctx context.Context) ([]*ledger.AccountingPeriod, error)
}

// AccountLookup resolves TK codes for the R7 account checks and lists the
// seeded COA for the step-3 preview.
type AccountLookup interface {
	GetAccountByCode(ctx context.Context, code string) (*ledger.Account, error)
	ListAccounts(ctx context.Context, f ledger.AccountFilter) ([]*ledger.Account, error)
}

// PostingLister lists posted journal entries for the R12 reopen guard.
type PostingLister interface {
	ListEntries(ctx context.Context, f ledger.EntryFilter) ([]*ledger.JournalEntry, error)
}

// Auditor appends the audit trail on every mutation (R13) and exposes the
// recent trail for the dashboard view.
type Auditor interface {
	Record(ctx context.Context, l *audit.AuditLog) error
	ListRecent(ctx context.Context, module string, limit int) ([]*audit.AuditLog, error)
}

type service struct {
	repo setup.Repository
	deps Dependencies
	mu   sync.Mutex
	now  func() time.Time
}

// NewService builds the setup orchestrator over the repository and seams.
func NewService(repo setup.Repository, deps Dependencies) Service {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &service{repo: repo, deps: deps, now: now}
}

// objectRequiredAccounts maps the TKs that MUST carry đối tượng detail to the
// masterdata kind used to validate the object (R10, docs/setup §2.3). In v1 only
// 131 (khách hàng) is mandatory; the other đối tượng TKs (331/152/155/156/
// 211/214) are validated whenever an object is present.
var objectRequiredAccounts = map[string]masterdata.Kind{
	"131": masterdata.KindCustomer,
}

// objectDetailAccounts maps the TKs that carry optional đối tượng detail.
var objectDetailAccounts = map[string]masterdata.Kind{
	"331": masterdata.KindSupplier,
	"152": masterdata.KindItem,
	"155": masterdata.KindItem,
	"156": masterdata.KindItem,
	"211": masterdata.KindFixedAssetCat,
	"214": masterdata.KindFixedAssetCat,
}

// --- reads --------------------------------------------------------------------

func (s *service) Status(ctx context.Context) (*StatusView, error) {
	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	view := &StatusView{Status: st, Steps: buildSteps(st)}
	profile, err := s.repo.GetProfile(ctx, setup.ProfileID)
	if err == nil {
		view.Profile = profile
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if s.deps.Regime != nil {
		view.Regime, _ = s.deps.Regime.GetRegime(ctx)
	}
	if st.Index() >= setup.StatusBalancesDraft.Index() {
		check, err := s.check(ctx)
		if err != nil {
			return nil, err
		}
		view.Check = check
		view.Balances, _ = s.repo.ListBalances(ctx, "")
	}
	return view, nil
}

func buildSteps(st setup.SetupStatus) []StatusStep {
	type step struct {
		key, label string
	}
	steps := []step{
		{"profile", "Thông tin doanh nghiệp"},
		{"regime", "Chế độ kế toán"},
		{"accounts", "Sơ đồ tài khoản"},
		{"periods", "Mở kỳ kế toán"},
		{"balances", "Số dư đầu kỳ"},
		{"lock", "Khóa số dư"},
		{"active", "Kích hoạt"},
	}
	out := make([]StatusStep, 0, len(steps))
	for i, stp := range steps {
		out = append(out, StatusStep{
			Key:   stp.key,
			Label: stp.label,
			Done:  st.Index() > i, // status index > step index means completed
		})
	}
	return out
}

func (s *service) GetProfile(ctx context.Context) (*setup.CompanyProfile, error) {
	return s.repo.GetProfile(ctx, setup.ProfileID)
}

// --- initialize (P1) ----------------------------------------------------------

func (s *service) Initialize(ctx context.Context, req *InitializeRequest, actor string) (*StatusView, error) {
	if req == nil || req.Profile == nil {
		return nil, setup.ErrInvalidProfile
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	if st.Index() >= setup.StatusBalancesLocked.Index() {
		return nil, setup.ErrAlreadyInitialized
	}

	// 1. Profile (empty → PROFILED), R1–R5.
	if st.Index() < setup.StatusProfiled.Index() {
		p := req.Profile.Clone()
		p.ID = setup.ProfileID
		regime := req.Regime
		if regime == "" {
			regime = p.AccountingRegime
		}
		if err := validateProfile(p, regime); err != nil {
			return nil, err
		}
		now := s.now().UTC().Format(time.RFC3339)
		p.Status = setup.StatusProfiled
		p.CreatedBy, p.CreatedAt = actor, now
		p.UpdatedBy, p.UpdatedAt = actor, now
		if err := s.repo.SaveProfile(ctx, p); err != nil {
			return nil, err
		}
		st = setup.StatusProfiled
		if err := s.repo.SetStatus(ctx, st); err != nil {
			return nil, err
		}
		s.audit(ctx, actor, "initialize.profile", setup.ProfileID)
	}

	// 2. Regime (PROFILED → REGIME_SET). masterdata.SetRegime validates again.
	if st.Index() < setup.StatusRegimeSet.Index() {
		regime := req.Regime
		if regime == "" {
			profile, err := s.repo.GetProfile(ctx, setup.ProfileID)
			if err != nil {
				return nil, err
			}
			regime = profile.AccountingRegime
		}
		if !setup.SupportedRegimes[strings.ToUpper(strings.TrimSpace(regime))] {
			return nil, setup.ErrInvalidRegime
		}
		if err := s.deps.Regime.SetRegime(ctx, strings.ToUpper(strings.TrimSpace(regime)), actor); err != nil {
			return nil, err
		}
		st = setup.StatusRegimeSet
		if err := s.repo.SetStatus(ctx, st); err != nil {
			return nil, err
		}
		s.audit(ctx, actor, "initialize.regime", regime)
	}

	// 3. Accounts (REGIME_SET → ACCOUNTS_SEEDED).
	if req.SeedAccounts && st.Index() < setup.StatusAccountsSeeded.Index() {
		if _, err := s.deps.Seeder.SeedAccounts(ctx, actor); err != nil {
			return nil, err
		}
		st = setup.StatusAccountsSeeded
		if err := s.repo.SetStatus(ctx, st); err != nil {
			return nil, err
		}
		s.audit(ctx, actor, "initialize.accounts", setup.ProfileID)
	}

	// 4. Periods (ACCOUNTS_SEEDED → PERIODS_OPEN), 12 monthly periods.
	if req.OpenPeriods && st.Index() < setup.StatusPeriodsOpen.Index() {
		fyStr := req.FiscalYearStart
		if fyStr == "" {
			profile, err := s.repo.GetProfile(ctx, setup.ProfileID)
			if err != nil {
				return nil, err
			}
			fyStr = profile.FiscalYearStart
		}
		fy, err := time.Parse("2006-01-02", fyStr)
		if err != nil {
			return nil, setup.ErrInvalidFiscalYear
		}
		for m := 1; m <= 12; m++ {
			period := fmt.Sprintf("%04d-%02d", fy.Year(), m)
			if _, err := s.deps.Periods.OpenPeriod(ctx, actor, period); err != nil {
				return nil, err
			}
		}
		st = setup.StatusPeriodsOpen
		if err := s.repo.SetStatus(ctx, st); err != nil {
			return nil, err
		}
		s.audit(ctx, actor, "initialize.periods", setup.ProfileID)
	}

	// 5. Ready for balances (PERIODS_OPEN → BALANCES_DRAFT).
	if st == setup.StatusPeriodsOpen {
		st = setup.StatusBalancesDraft
		if err := s.repo.SetStatus(ctx, st); err != nil {
			return nil, err
		}
	}
	return s.Status(ctx)
}

func (s *service) UpdateProfile(ctx context.Context, p *setup.CompanyProfile, actor string) (*setup.CompanyProfile, error) {
	if p == nil {
		return nil, setup.ErrInvalidProfile
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	if st.Index() >= setup.StatusBalancesLocked.Index() {
		return nil, setup.ErrWrongState
	}
	existing, err := s.repo.GetProfile(ctx, setup.ProfileID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err := validateProfile(p, p.AccountingRegime); err != nil {
		return nil, err
	}
	now := s.now().UTC().Format(time.RFC3339)
	p.ID = setup.ProfileID
	p.Status = setup.StatusProfiled
	if existing != nil {
		p.CreatedBy, p.CreatedAt = existing.CreatedBy, existing.CreatedAt
	}
	p.UpdatedBy, p.UpdatedAt = actor, now
	if err := s.repo.SaveProfile(ctx, p); err != nil {
		return nil, err
	}
	s.audit(ctx, actor, "profile.update", setup.ProfileID)
	return p.Clone(), nil
}

// validateProfile enforces R1–R5 + statutory fields on the profile.
func validateProfile(p *setup.CompanyProfile, regime string) error {
	switch {
	case strings.TrimSpace(p.Name) == "":
		return &setup.ValidationError{Field: "name", MessageVn: "Tên doanh nghiệp không được để trống", MessageEn: "name is required"}
	case !masterdata.ValidTaxCode(p.TaxCode):
		return &setup.ValidationError{Field: "tax_code", MessageVn: "MST không hợp lệ (10 hoặc 13 số)", MessageEn: "invalid MST (tax code)"}
	case strings.TrimSpace(p.Address) == "":
		return &setup.ValidationError{Field: "address", MessageVn: "Địa chỉ không được để trống", MessageEn: "address is required"}
	case strings.TrimSpace(p.LegalRepresentative) == "":
		return &setup.ValidationError{Field: "legal_representative", MessageVn: "Người đại diện không được để trống", MessageEn: "legal representative is required"}
	}
	cur := strings.ToUpper(strings.TrimSpace(p.AccountingCurrency))
	if cur == "" {
		cur = "VND"
	}
	if cur != "VND" {
		return &setup.ValidationError{Field: "accounting_currency", MessageVn: "Tiền tệ hạch toán phải là VND (v1)", MessageEn: "accounting currency must be VND in v1"}
	}
	if p.FiscalYearStart != "" {
		t, err := time.Parse("2006-01-02", p.FiscalYearStart)
		if err != nil || t.Day() != 1 {
			return &setup.ValidationError{Field: "fiscal_year_start", MessageVn: "Năm tài chính phải bắt đầu ngày 01 của một tháng", MessageEn: "fiscal year must start on the 1st of a month"}
		}
	}
	if regime == "" {
		regime = p.AccountingRegime
	}
	if !setup.SupportedRegimes[strings.ToUpper(strings.TrimSpace(regime))] {
		return &setup.ValidationError{Field: "accounting_regime", MessageVn: "Chưa hỗ trợ chế độ kế toán này", MessageEn: "unsupported accounting regime"}
	}
	if p.AccountingRegime != "" && !setup.SupportedRegimes[strings.ToUpper(strings.TrimSpace(p.AccountingRegime))] {
		return &setup.ValidationError{Field: "accounting_regime", MessageVn: "Chưa hỗ trợ chế độ kế toán này", MessageEn: "unsupported accounting regime"}
	}
	return nil
}

// --- opening balances (P2) ----------------------------------------------------

func (s *service) SaveBalance(ctx context.Context, b *setup.OpeningBalance, actor string) (*setup.OpeningBalance, error) {
	if b == nil {
		return nil, setup.ErrInvalidBalance
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	switch {
	case st == setup.StatusBalancesLocked || st == setup.StatusActive:
		return nil, setup.ErrBalanceLocked
	case st.Index() < setup.StatusBalancesDraft.Index():
		return nil, setup.ErrWrongState
	}

	if err := s.validateBalance(ctx, b); err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfile(ctx, setup.ProfileID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC().Format(time.RFC3339)
	b.ID = setup.BalanceID(b.AccountCode, b.ObjectCode)
	b.Period = core.Period{From: profile.FiscalYearStart, To: profile.FiscalYearStart}
	b.Debit.Currency = "VND"
	b.Credit.Currency = "VND"
	b.Status = setup.BalanceDraft
	if b.EnteredAt == "" {
		b.EnteredBy, b.EnteredAt = actor, now
	}
	b.UpdatedBy, b.UpdatedAt = actor, now
	if err := s.repo.SaveBalance(ctx, b); err != nil {
		return nil, err
	}
	s.audit(ctx, actor, "balance.upsert", b.ID)
	return b.Clone(), nil
}

// validateBalance enforces R7 (one side), R8 (draft handled by caller) and
// R10 (account + đối tượng rules) plus the account existence checks.
func (s *service) validateBalance(ctx context.Context, b *setup.OpeningBalance) error {
	debit := b.Debit.AmountMinor
	credit := b.Credit.AmountMinor
	if debit < 0 || credit < 0 || (debit == 0) == (credit == 0) {
		return setup.ErrInvalidBalance
	}

	acct, err := s.deps.Accounts.GetAccountByCode(ctx, b.AccountCode)
	if err != nil {
		return setup.ErrAccountNotFound
	}
	if acct.Status != ledger.AccountActive || !acct.AllowPost {
		return setup.ErrAccountNotFound
	}

	// R10: đối tượng detail — mandatory on 131, validated when present on the
	// other detail TKs (331/152/155/156/211/214).
	if kind, required := objectRequiredAccounts[b.AccountCode]; required {
		if b.ObjectCode == "" {
			return setup.ErrObjectRequired
		}
		if b.ObjectType != "" && b.ObjectType != string(kind) {
			return setup.ErrObjectNotFound
		}
		obj, err := s.deps.Objects.Get(ctx, kind, b.ObjectCode)
		if err != nil || obj.State != masterdata.StateActive {
			return setup.ErrObjectNotFound
		}
	} else if kind, ok := objectDetailAccounts[b.AccountCode]; ok && b.ObjectCode != "" {
		if b.ObjectType != "" && b.ObjectType != string(kind) {
			return setup.ErrObjectNotFound
		}
		obj, err := s.deps.Objects.Get(ctx, kind, b.ObjectCode)
		if err != nil || obj.State != masterdata.StateActive {
			return setup.ErrObjectNotFound
		}
	}
	return nil
}

func (s *service) ListBalances(ctx context.Context, accountCode string) (*BalanceList, error) {
	balances, err := s.repo.ListBalances(ctx, accountCode)
	if err != nil {
		return nil, err
	}
	check := &setup.BalanceCheck{}
	if accountCode == "" {
		check, err = s.check(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &BalanceList{Balances: balances, Check: check}, nil
}

func (s *service) DeleteBalance(ctx context.Context, id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st == setup.StatusBalancesLocked || st == setup.StatusActive {
		return setup.ErrBalanceLocked
	}
	if st.Index() < setup.StatusBalancesDraft.Index() {
		return setup.ErrWrongState
	}
	if err := s.repo.DeleteBalance(ctx, id); err != nil {
		return err
	}
	s.audit(ctx, actor, "balance.delete", id)
	return nil
}

func (s *service) CheckBalances(ctx context.Context) (*setup.BalanceCheck, error) {
	return s.check(ctx)
}

func (s *service) check(ctx context.Context) (*setup.BalanceCheck, error) {
	balances, err := s.repo.ListBalances(ctx, "")
	if err != nil {
		return nil, err
	}
	out := &setup.BalanceCheck{}
	perAccount := map[string][2]int64{} // code -> {debit, credit}
	for _, b := range balances {
		out.Debit += b.Debit.AmountMinor
		out.Credit += b.Credit.AmountMinor
		p := perAccount[b.AccountCode]
		p[0] += b.Debit.AmountMinor
		p[1] += b.Credit.AmountMinor
		perAccount[b.AccountCode] = p
	}
	for _, b := range balances {
		if _, required := objectRequiredAccounts[b.AccountCode]; required && b.ObjectCode == "" {
			out.Gaps = append(out.Gaps, b.AccountCode)
		}
	}
	out.Diff = out.Debit - out.Credit
	// An empty balance sheet is not "balanced": Lock must not succeed before
	// any opening balances are entered.
	out.Balanced = len(balances) > 0 && out.Diff == 0 && len(out.Gaps) == 0
	// Offending: the accounts carrying the surplus side, so the operator knows
	// which TKs to adjust — empty when the sheet is balanced (a TK holding a
	// natural debit or credit balance is not itself an error).
	if !out.Balanced && out.Diff != 0 {
		surplusDebit := out.Diff > 0
		for code, p := range perAccount {
			net := p[0] - p[1]
			if (surplusDebit && net > 0) || (!surplusDebit && net < 0) {
				out.Offending = append(out.Offending, code)
			}
		}
	}
	return out, nil
}

// --- lock / reopen / activate (P2) ---------------------------------------------

func (s *service) Lock(ctx context.Context, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st != setup.StatusBalancesDraft {
		return setup.ErrWrongState
	}
	check, err := s.check(ctx)
	if err != nil {
		return err
	}
	if !check.Balanced {
		return setup.ErrUnbalanced
	}
	if err := s.repo.SetStatus(ctx, setup.StatusBalancesLocked); err != nil {
		return err
	}
	s.audit(ctx, actor, "balances.lock", setup.ProfileID)
	return nil
}

func (s *service) Reopen(ctx context.Context, actor, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return setup.ErrReopenBlocked
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st != setup.StatusBalancesLocked {
		return setup.ErrWrongState
	}
	// R12: blocked while a POSTED voucher references any balance TK.
	balances, err := s.repo.ListBalances(ctx, "")
	if err != nil {
		return err
	}
	codes := map[string]bool{}
	for _, b := range balances {
		codes[b.AccountCode] = true
	}
	if len(codes) > 0 {
		posted, err := s.deps.Postings.ListEntries(ctx, ledger.EntryFilter{Status: ledger.EntryPosted})
		if err != nil {
			return err
		}
		for _, e := range posted {
			for _, line := range e.Lines {
				if codes[line.AccountCode] {
					return setup.ErrReopenBlocked
				}
			}
		}
	}
	if err := s.repo.SetStatus(ctx, setup.StatusBalancesDraft); err != nil {
		return err
	}
	s.audit(ctx, actor, "balances.reopen", setup.ProfileID)
	return nil
}

func (s *service) Activate(ctx context.Context, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st != setup.StatusBalancesLocked {
		return setup.ErrWrongState
	}
	if err := s.repo.SetStatus(ctx, setup.StatusActive); err != nil {
		return err
	}
	s.audit(ctx, actor, "activate", setup.ProfileID)
	return nil
}

// --- CSV import (P3) -----------------------------------------------------------

// importColumns is the v1 CSV template header (docs/setup §5.4).
var importColumns = []string{"account", "object_type", "object_code", "debit", "credit"}

// importHeaderOK rejects any CSV whose header is not exactly template v1
// (T3.1 — template version rejection). Column order matters; the parser maps
// positions, so a shuffled header would silently corrupt data.
func importHeaderOK(header []string) bool {
	if len(header) != len(importColumns) {
		return false
	}
	for i, want := range importColumns {
		if strings.ReplaceAll(strings.ToLower(strings.TrimSpace(header[i])), " ", "") != want {
			return false
		}
	}
	return true
}

// importJobID derives the deterministic job id for one upload: sha256 of the
// CSV content (header included) plus the dry-run flag, so re-uploading the
// same file overwrites its own report and dry-run/commit reports never
// collide.
func importJobID(rows [][]string, dryRun bool) string {
	h := sha256.New()
	for _, r := range rows {
		h.Write([]byte(strings.Join(r, "\x1f")))
		h.Write([]byte{0})
	}
	h.Write([]byte(strconv.FormatBool(dryRun)))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *service) ImportBalances(ctx context.Context, rows [][]string, actor string, dryRun bool) (*setup.ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.repo.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	if st == setup.StatusBalancesLocked || st == setup.StatusActive {
		return nil, setup.ErrBalanceLocked
	}
	if st.Index() < setup.StatusBalancesDraft.Index() {
		return nil, setup.ErrWrongState
	}

	if len(rows) == 0 || !importHeaderOK(rows[0]) {
		return nil, setup.ErrInvalidImport
	}
	rows = rows[1:] // header

	res := &setup.ImportResult{DryRun: dryRun, Total: len(rows)}

	profile, err := s.repo.GetProfile(ctx, setup.ProfileID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC().Format(time.RFC3339)
	existing := map[string]bool{}
	if !dryRun {
		all, err := s.repo.ListBalances(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, b := range all {
			existing[b.ID] = true
		}
	}

	var pending []*setup.OpeningBalance
	for i, row := range rows {
		line := i + 2 // 1-based with header
		b, err := parseBalanceRow(row)
		if err != nil {
			res.Errors = append(res.Errors, setup.RowError{Row: line, Message: err.Error()})
			continue
		}
		if err := s.validateBalance(ctx, b); err != nil {
			res.Errors = append(res.Errors, setup.RowError{Row: line, Message: err.Error()})
			continue
		}
		if dryRun {
			continue
		}
		b.ID = setup.BalanceID(b.AccountCode, b.ObjectCode)
		b.Period = core.Period{From: profile.FiscalYearStart, To: profile.FiscalYearStart}
		b.Debit.Currency = "VND"
		b.Credit.Currency = "VND"
		b.Status = setup.BalanceDraft
		b.EnteredBy, b.EnteredAt = actor, now
		b.UpdatedBy, b.UpdatedAt = actor, now
		if existing[b.ID] {
			res.Updated++
		} else {
			res.Created++
			existing[b.ID] = true
		}
		pending = append(pending, b)
	}
	if len(pending) > 0 {
		// one tx per batch (spec §5.4): all-or-nothing.
		if err := s.repo.SaveBalances(ctx, pending); err != nil {
			return nil, err
		}
	}

	res.JobID = importJobID(append([][]string{importColumns}, rows...), dryRun)
	jobStatus := setup.JobOK
	if len(res.Errors) > 0 {
		jobStatus = setup.JobErrored
	}
	job := &setup.ImportJob{
		ID: res.JobID, Status: jobStatus, Total: res.Total,
		Created: res.Created, Updated: res.Updated, Errors: res.Errors,
		DryRun: dryRun, CreatedBy: actor, CreatedAt: now,
	}
	if err := s.repo.SaveImportJob(ctx, job); err != nil {
		return nil, err
	}
	s.audit(ctx, actor, "balances.import", res.JobID)
	return res, nil
}

func (s *service) GetImportReport(ctx context.Context, jobID string) (*setup.ImportJob, error) {
	return s.repo.GetImportJob(ctx, jobID)
}

// AuditTrail returns the recent setup-module audit trail for the dashboard
// (R13): profile saves, seed, period opens, balance mutations, imports,
// locks/reopens and activation.
func (s *service) AuditTrail(ctx context.Context, module string, limit int) ([]*audit.AuditLog, error) {
	return s.deps.Audit.ListRecent(ctx, module, limit)
}

// PreviewAccounts lists the seeded COA for the wizard step 3 ("Tài khoản").
// Read-only: no status guard, so the page is safe to render at any point.
func (s *service) PreviewAccounts(ctx context.Context) (*setup.AccountPreview, error) {
	accts, err := s.deps.Accounts.ListAccounts(ctx, ledger.AccountFilter{})
	if err != nil {
		return nil, err
	}
	out := &setup.AccountPreview{Total: len(accts)}
	for _, a := range accts {
		out.Accounts = append(out.Accounts, setup.PreviewAccount{
			Code: a.Code, Name: a.Name, Type: string(a.Type), Postable: a.AllowPost,
		})
	}
	return out, nil
}

// parseBalanceRow maps one CSV data row onto an OpeningBalance draft.
func parseBalanceRow(row []string) (*setup.OpeningBalance, error) {
	if len(row) < 5 {
		return nil, setup.ErrInvalidBalance
	}
	acct := strings.TrimSpace(row[0])
	objectType := strings.TrimSpace(row[1])
	objectCode := strings.TrimSpace(row[2])
	debit, err := parseAmount(row[3])
	if err != nil {
		return nil, setup.ErrInvalidBalance
	}
	credit, err := parseAmount(row[4])
	if err != nil {
		return nil, setup.ErrInvalidBalance
	}
	return &setup.OpeningBalance{
		AccountCode: acct,
		ObjectType:  objectType,
		ObjectCode:  objectCode,
		Debit:       core.Money{AmountMinor: debit, Currency: "VND"},
		Credit:      core.Money{AmountMinor: credit, Currency: "VND"},
	}, nil
}

func parseAmount(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// --- audit (R13) ---------------------------------------------------------------

func (s *service) audit(ctx context.Context, actor, action, target string) {
	if s.deps.Audit == nil {
		return
	}
	_ = s.deps.Audit.Record(ctx, &audit.AuditLog{
		UserCode: actor,
		Module:   "setup",
		Action:   action,
		TargetID: target,
	})
}
