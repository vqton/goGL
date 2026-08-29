package fixedasset

import (
	"context"
	"fmt"

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
}

type service struct {
	repo fixedasset.Repository
}

func NewService(repo fixedasset.Repository) Service {
	return &service{repo: repo}
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
