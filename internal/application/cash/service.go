package cash

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	domainaudit "goGL/internal/domain/audit"
	"goGL/internal/domain/cash"
)

// AuditRecorder is the seam the cash service uses to write audit rows. The
// audit module's Service satisfies it; swap freely without coupling.
type AuditRecorder interface {
	Record(ctx context.Context, l *domainaudit.AuditLog) error
}

type Service interface {
	CreateFund(ctx context.Context, f *cash.Fund) error
	ListFunds(ctx context.Context) ([]*cash.Fund, error)

	CreateVoucher(ctx context.Context, actor string, v *cash.Voucher) error
	UpdateVoucher(ctx context.Context, actor string, v *cash.Voucher) error
	ApproveVoucher(ctx context.Context, actor, id string) (*cash.Voucher, error)
	GetVoucher(ctx context.Context, id string) (*cash.Voucher, error)
	ListVouchers(ctx context.Context, f cash.VoucherFilter) ([]*cash.Voucher, error)
}

type service struct {
	repo  cash.Repository
	audit AuditRecorder
	now   func() time.Time
}

func NewService(repo cash.Repository, audit AuditRecorder) Service {
	return &service{repo: repo, audit: audit, now: time.Now}
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *service) auditAction(ctx context.Context, actor, action, targetID string) error {
	return s.audit.Record(ctx, &domainaudit.AuditLog{
		UserCode:  actor,
		Module:    "cash",
		Action:    action,
		TargetID:  targetID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) CreateFund(ctx context.Context, f *cash.Fund) error {
	if f.Name == "" || f.Currency == "" || f.Account == "" {
		return errors.New("cash: fund name, currency and account are required")
	}
	if f.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		f.ID = id
	}
	if err := s.repo.CreateFund(ctx, f); err != nil {
		return err
	}
	return s.auditAction(ctx, "", "fund.create", f.ID)
}

func (s *service) ListFunds(ctx context.Context) ([]*cash.Fund, error) {
	return s.repo.ListFunds(ctx)
}

func (s *service) CreateVoucher(ctx context.Context, actor string, v *cash.Voucher) error {
	fund, err := s.loadActiveFund(ctx, v.FundID)
	if err != nil {
		return err
	}
	if v.Currency == "" {
		v.Currency = fund.Currency
	}
	if v.Currency != fund.Currency {
		return errors.New("cash: voucher currency must match the fund currency")
	}
	if err := validateVoucher(v, fund.Account); err != nil {
		return err
	}

	id, err := newID()
	if err != nil {
		return err
	}
	v.ID = id
	v.State = cash.VoucherDraft
	v.CreatedBy = actor
	v.ApprovedBy = ""
	v.PostedBy = ""
	v.ApprovedAt = ""
	v.PostedAt = ""
	v.AmountWords = cash.AmountInWords(v.AmountMinor)

	refNo, err := s.repo.NextRefNo(ctx, v.FundID, v.RefDate[:7], v.Type)
	if err != nil {
		return err
	}
	v.RefNo = refNo

	if err := s.repo.CreateVoucher(ctx, v); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "voucher.create", v.ID)
}

func (s *service) UpdateVoucher(ctx context.Context, actor string, v *cash.Voucher) error {
	existing, err := s.repo.GetVoucher(ctx, v.ID)
	if err != nil {
		return err
	}
	if existing.State != cash.VoucherDraft {
		return cash.ErrWrongState
	}
	if v.FundID != existing.FundID || v.Type != existing.Type {
		return errors.New("cash: fund and type are immutable after creation")
	}

	fund, err := s.loadActiveFund(ctx, v.FundID)
	if err != nil {
		return err
	}
	if v.Currency == "" {
		v.Currency = fund.Currency
	}
	if v.Currency != fund.Currency {
		return errors.New("cash: voucher currency must match the fund currency")
	}
	if err := validateVoucher(v, fund.Account); err != nil {
		return err
	}

	v.ID = existing.ID
	v.RefNo = existing.RefNo
	v.CreatedBy = existing.CreatedBy
	v.State = cash.VoucherDraft
	v.ApprovedBy = ""
	v.PostedBy = ""
	v.ApprovedAt = ""
	v.PostedAt = ""
	v.AmountWords = cash.AmountInWords(v.AmountMinor)

	if err := s.repo.UpdateVoucher(ctx, v); err != nil {
		return err
	}
	return s.auditAction(ctx, actor, "voucher.update", v.ID)
}

func (s *service) ApproveVoucher(ctx context.Context, actor, id string) (*cash.Voucher, error) {
	v, err := s.repo.GetVoucher(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.State != cash.VoucherDraft {
		return nil, cash.ErrWrongState
	}
	if actor == v.CreatedBy {
		return nil, cash.ErrSelfApproval
	}
	v.State = cash.VoucherApproved
	v.ApprovedBy = actor
	v.ApprovedAt = s.now().UTC().Format(time.RFC3339)

	if err := s.repo.UpdateVoucher(ctx, v); err != nil {
		return nil, err
	}
	if err := s.auditAction(ctx, actor, "voucher.approve", id); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *service) GetVoucher(ctx context.Context, id string) (*cash.Voucher, error) {
	return s.repo.GetVoucher(ctx, id)
}

func (s *service) ListVouchers(ctx context.Context, f cash.VoucherFilter) ([]*cash.Voucher, error) {
	return s.repo.ListVouchers(ctx, f)
}

func (s *service) loadActiveFund(ctx context.Context, id string) (*cash.Fund, error) {
	fund, err := s.repo.GetFund(ctx, id)
	if err != nil {
		return nil, cash.ErrFundNotFound
	}
	if !fund.IsActive {
		return nil, cash.ErrFundInactive
	}
	return fund, nil
}

func validateVoucher(v *cash.Voucher, cashAccount string) error {
	if v.Type != cash.VoucherReceive && v.Type != cash.VoucherPay {
		return errors.New("cash: invalid voucher type")
	}
	if v.FundID == "" {
		return errors.New("cash: fund_id is required")
	}
	if v.RefDate == "" || len(v.RefDate) < 7 {
		return errors.New("cash: ref_date is required as yyyy-mm-dd")
	}
	if v.AmountMinor <= 0 {
		return errors.New("cash: amount must be positive")
	}
	if v.CounterpartyName == "" {
		return errors.New("cash: counterparty name is required (R1)")
	}
	if len(v.Lines) < 2 {
		return cash.ErrInvalidLines
	}

	var sumDebit, sumCredit int64
	cashLines := 0
	for _, l := range v.Lines {
		if l.AmountMinor <= 0 {
			return cash.ErrInvalidLines
		}
		if l.DebitAcc != "" {
			sumDebit += l.AmountMinor
		}
		if l.CreditAcc != "" {
			sumCredit += l.AmountMinor
		}
		if strings.HasPrefix(l.DebitAcc, cashAccount) || strings.HasPrefix(l.CreditAcc, cashAccount) {
			cashLines++
		}
	}
	if sumDebit != v.AmountMinor || sumCredit != v.AmountMinor {
		return cash.ErrInvalidLines
	}
	if cashLines != 1 {
		return cash.ErrInvalidLines
	}
	return nil
}
