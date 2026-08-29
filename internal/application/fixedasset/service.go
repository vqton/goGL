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
	List(ctx context.Context, assetType fixedasset.AssetType, state fixedasset.AssetState) ([]*fixedasset.FixedAsset, error)
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

func (s *service) List(ctx context.Context, assetType fixedasset.AssetType, state fixedasset.AssetState) ([]*fixedasset.FixedAsset, error) {
	out, err := s.repo.List(ctx, assetType, state)
	if err != nil {
		return nil, fmt.Errorf("fixedasset: list: %w", err)
	}
	return out, nil
}
