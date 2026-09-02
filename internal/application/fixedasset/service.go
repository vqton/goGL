package fixedasset

import (
	"context"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/fixedasset"
)

// Service defines the business operations for fixed assets.
type Service interface {
	Create(ctx context.Context, a *fixedasset.FixedAsset) error
	GetByID(ctx context.Context, id string) (*fixedasset.FixedAsset, error)
	Update(ctx context.Context, a *fixedasset.FixedAsset) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, assetType fixedasset.AssetType, state fixedasset.AssetState, limit, offset int) ([]*fixedasset.FixedAsset, error)
	Transfer(ctx context.Context, id, location, department string) (*fixedasset.FixedAsset, error)
	Liquidate(ctx context.Context, id string) (*fixedasset.FixedAsset, error)
	ConfirmLiquidation(ctx context.Context, id string, newState fixedasset.AssetState) (*fixedasset.FixedAsset, error)
	Deactivate(ctx context.Context, id string) (*fixedasset.FixedAsset, error)
	Reactivate(ctx context.Context, id string) (*fixedasset.FixedAsset, error)

	// Depreciation
	RunMonthlyDepreciation(ctx context.Context, period, actor string) ([]*fixedasset.DepreciationEntry, error)
	GetDepreciationSchedule(ctx context.Context, assetID string) ([]*fixedasset.DepreciationEntry, error)
	GetDepreciationByPeriod(ctx context.Context, period string) ([]*fixedasset.DepreciationEntry, error)

	// Approval workflow
	ApproveLiquidation(ctx context.Context, id, actor string) (*fixedasset.FixedAsset, error)
	RejectLiquidation(ctx context.Context, id, actor, reason string) (*fixedasset.FixedAsset, error)
}

type service struct {
	repo     fixedasset.Repository
	deprRepo fixedasset.DepreciationEntryRepository
	engine   *DepreciationEngine
}

func NewService(repo fixedasset.Repository) Service {
	return &service{repo: repo, engine: NewDepreciationEngine()}
}

func NewServiceWithDepreciation(repo fixedasset.Repository, deprRepo fixedasset.DepreciationEntryRepository) Service {
	return &service{repo: repo, deprRepo: deprRepo, engine: NewDepreciationEngine()}
}

func (s *service) Create(ctx context.Context, a *fixedasset.FixedAsset) error {
	if err := fixedasset.ValidateAsset(a); err != nil {
		return err
	}
	seq, err := s.repo.NextCode(ctx)
	if err != nil {
		return fmt.Errorf("fixedasset: next code: %w", err)
	}
	a.Code = fmt.Sprintf("FA-%05d", seq)
	a.State = fixedasset.StateActive
	a.CreatedAt = fixedasset.NowRFC3339()
	a.UpdatedAt = a.CreatedAt
	if err := s.repo.Create(ctx, a); err != nil {
		return fmt.Errorf("fixedasset: create: %w", err)
	}
	return nil
}

func (s *service) GetByID(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: get: %w", err)
	}
	return a, nil
}

func (s *service) Update(ctx context.Context, a *fixedasset.FixedAsset) error {
	existing, err := s.repo.FindByID(ctx, a.ID)
	if err != nil {
		return fmt.Errorf("fixedasset: update lookup: %w", err)
	}
	a.Code = existing.Code
	a.CreatedAt = existing.CreatedAt
	a.CreatedBy = existing.CreatedBy
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return fmt.Errorf("fixedasset: update: %w", err)
	}
	return nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fixedasset: delete lookup: %w", err)
	}
	if a.State == fixedasset.StateActive {
		return fixedasset.ErrConflict
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("fixedasset: delete: %w", err)
	}
	return nil
}

func (s *service) List(ctx context.Context, assetType fixedasset.AssetType, state fixedasset.AssetState, limit, offset int) ([]*fixedasset.FixedAsset, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	out, err := s.repo.List(ctx, assetType, state, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: list: %w", err)
	}
	return out, nil
}

func (s *service) Transfer(ctx context.Context, id, location, department string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: transfer lookup: %w", err)
	}
	if a.State != fixedasset.StateActive {
		return nil, fixedasset.ErrConflict
	}
	a.Location = location
	a.Department = department
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: transfer update: %w", err)
	}
	return a, nil
}

func (s *service) Liquidate(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: liquidate lookup: %w", err)
	}
	if a.State == fixedasset.StateScrapped || a.State == fixedasset.StateSold {
		return nil, fixedasset.ErrConflict
	}
	a.State = fixedasset.StatePendingLiquidation
	a.ApprovalStatus = fixedasset.ApprovalPending
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: liquidate update: %w", err)
	}
	return a, nil
}

func (s *service) ConfirmLiquidation(ctx context.Context, id string, newState fixedasset.AssetState) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: confirm lookup: %w", err)
	}
	if a.State != fixedasset.StatePendingLiquidation {
		return nil, fixedasset.ErrConflict
	}
	if newState != fixedasset.StateScrapped && newState != fixedasset.StateSold {
		return nil, fixedasset.ErrInvalid
	}
	a.State = newState
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: confirm update: %w", err)
	}
	return a, nil
}

func (s *service) Deactivate(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: deactivate lookup: %w", err)
	}
	if a.State != fixedasset.StateActive {
		return nil, fixedasset.ErrConflict
	}
	a.State = fixedasset.StateInactive
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: deactivate update: %w", err)
	}
	return a, nil
}

func (s *service) Reactivate(ctx context.Context, id string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: reactivate lookup: %w", err)
	}
	if a.State != fixedasset.StateInactive {
		return nil, fixedasset.ErrConflict
	}
	a.State = fixedasset.StateActive
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: reactivate update: %w", err)
	}
	return a, nil
}

// --- Depreciation methods ---

func (s *service) RunMonthlyDepreciation(ctx context.Context, period, actor string) ([]*fixedasset.DepreciationEntry, error) {
	if s.deprRepo == nil {
		return nil, fixedasset.ErrInvalid
	}

	posted, err := s.deprRepo.IsPeriodPosted(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("depreciation: check period: %w", err)
	}
	if posted {
		return nil, fixedasset.ErrConflict
	}

	assets, err := s.repo.List(ctx, "", fixedasset.StateActive, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("depreciation: list assets: %w", err)
	}

	var entries []*fixedasset.DepreciationEntry
	now := fixedasset.NowRFC3339()

	for _, a := range assets {
		if a.InServiceDate == "" {
			continue
		}

		amount, err := s.engine.MonthlyDepreciation(a)
		if err != nil {
			continue
		}
		if amount == 0 {
			continue
		}

		debitAccount := pickDebitAccount(a.Department)

		entry := &fixedasset.DepreciationEntry{
			ID:                 core.RowID("depr", a.ID, period),
			AssetID:            a.ID,
			AssetCode:          a.Code,
			AssetName:          a.Name,
			Period:             period,
			DepreciationMethod: a.DepreciationMethod,
			Amount:             amount,
			AccumulatedDepr:    a.AccumulatedDepr + amount,
			BookValue:          a.OriginalCost - a.AccumulatedDepr - amount,
			AccountDebit:       debitAccount,
			AccountCredit:      "2141",
			Status:             fixedasset.DepreciationDraft,
			CreatedBy:          actor,
			CreatedAt:          now,
		}
		entries = append(entries, entry)
	}

	for _, e := range entries {
		if err := s.deprRepo.Create(ctx, e); err != nil {
			return nil, fmt.Errorf("depreciation: create entry: %w", err)
		}
	}

	return entries, nil
}

func (s *service) GetDepreciationSchedule(ctx context.Context, assetID string) ([]*fixedasset.DepreciationEntry, error) {
	if s.deprRepo == nil {
		return nil, fixedasset.ErrInvalid
	}
	return s.deprRepo.ListByAsset(ctx, assetID)
}

func (s *service) GetDepreciationByPeriod(ctx context.Context, period string) ([]*fixedasset.DepreciationEntry, error) {
	if s.deprRepo == nil {
		return nil, fixedasset.ErrInvalid
	}
	return s.deprRepo.ListByPeriod(ctx, period)
}

// pickDebitAccount returns the appropriate depreciation expense account based on department.
func pickDebitAccount(department string) string {
	switch {
	case department == "" || department == "production" || department == "sản xuất":
		return "627"
	case department == "management" || department == "quản lý":
		return "641"
	case department == "commerce" || department == "thương mại":
		return "642"
	default:
		return "627"
	}
}

// --- Approval workflow ---

func (s *service) ApproveLiquidation(ctx context.Context, id, actor string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: approve lookup: %w", err)
	}
	if a.State != fixedasset.StatePendingLiquidation {
		return nil, fixedasset.ErrConflict
	}
	if a.ApprovalStatus != fixedasset.ApprovalPending {
		return nil, fixedasset.ErrConflict
	}
	a.ApprovalStatus = fixedasset.ApprovalApproved
	a.ApprovedBy = actor
	a.ApprovedAt = fixedasset.NowRFC3339()
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: approve update: %w", err)
	}
	return a, nil
}

func (s *service) RejectLiquidation(ctx context.Context, id, actor, reason string) (*fixedasset.FixedAsset, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: reject lookup: %w", err)
	}
	if a.State != fixedasset.StatePendingLiquidation {
		return nil, fixedasset.ErrConflict
	}
	if a.ApprovalStatus != fixedasset.ApprovalPending {
		return nil, fixedasset.ErrConflict
	}
	a.ApprovalStatus = fixedasset.ApprovalRejected
	a.ApprovedBy = actor
	a.ApprovedAt = fixedasset.NowRFC3339()
	a.RejectReason = reason
	a.State = fixedasset.StateActive // Revert to active on rejection
	a.UpdatedAt = fixedasset.NowRFC3339()
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("fixedasset: reject update: %w", err)
	}
	return a, nil
}
