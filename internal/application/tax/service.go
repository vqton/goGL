package tax

import (
	"context"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/tax"
)

type Service interface {
	Create(ctx context.Context, d *tax.TaxDeclaration, actor string) (*tax.TaxDeclaration, error)
	Get(ctx context.Context, id string) (*tax.TaxDeclaration, error)
	Update(ctx context.Context, id string, patch *tax.TaxDeclaration, actor string) (*tax.TaxDeclaration, error)
	List(ctx context.Context, taxType tax.TaxType, period string) ([]*tax.TaxDeclaration, error)
	File(ctx context.Context, id, actor string) (*tax.TaxDeclaration, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo tax.Repository
}

func NewService(repo tax.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, d *tax.TaxDeclaration, actor string) (*tax.TaxDeclaration, error) {
	decl := d.Clone()
	decl.State = tax.StateDraft
	decl.CreatedBy = actor
	decl.UpdatedBy = actor

	if err := tax.ValidateDeclaration(decl); err != nil {
		return nil, err
	}

	n, err := s.repo.NextCode(ctx, decl.TaxType)
	if err != nil {
		return nil, err
	}
	decl.Code = fmt.Sprintf("TAX-%s-%04d", tax.TaxTypeCode(decl.TaxType), n)
	decl.ID = core.RowID("tax", decl.Code)

	now := core.NowRFC3339()
	decl.CreatedAt = now
	decl.UpdatedAt = now
	decl.Recalculate()

	if err := s.repo.Create(ctx, decl); err != nil {
		return nil, err
	}
	return decl, nil
}

func (s *service) Get(ctx context.Context, id string) (*tax.TaxDeclaration, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, patch *tax.TaxDeclaration, actor string) (*tax.TaxDeclaration, error) {
	cur, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.State != tax.StateDraft {
		return nil, tax.ErrLocked
	}

	if patch.Period != "" {
		cur.Period = patch.Period
	}
	if patch.RefNo != "" {
		cur.RefNo = patch.RefNo
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}
	if patch.Items != nil {
		cur.Items = patch.Items
	}
	// tax_type is immutable after creation

	if err := tax.ValidateDeclaration(cur); err != nil {
		return nil, err
	}

	cur.Recalculate()
	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) List(ctx context.Context, taxType tax.TaxType, period string) ([]*tax.TaxDeclaration, error) {
	return s.repo.List(ctx, taxType, period)
}

func (s *service) File(ctx context.Context, id, actor string) (*tax.TaxDeclaration, error) {
	decl, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if decl.State != tax.StateDraft {
		return nil, tax.ErrLocked
	}

	decl.State = tax.StateFiled
	decl.FiledBy = actor
	decl.FiledAt = core.NowRFC3339()
	decl.UpdatedBy = actor
	decl.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, decl); err != nil {
		return nil, err
	}
	return decl, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	decl, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if decl.State != tax.StateDraft {
		return tax.ErrLocked
	}
	return s.repo.Delete(ctx, id)
}
