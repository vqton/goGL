package document

import (
	"context"
	"fmt"

	"goGL/internal/domain/core"
	"goGL/internal/domain/document"
)

type Service interface {
	Create(ctx context.Context, d *document.Document, actor string) (*document.Document, error)
	Get(ctx context.Context, id string) (*document.Document, error)
	Update(ctx context.Context, id string, patch *document.Document, actor string) (*document.Document, error)
	List(ctx context.Context, owner string, docType string, state string) ([]*document.Document, error)
	Archive(ctx context.Context, id, actor string) (*document.Document, error)
	Delete(ctx context.Context, id, actor string) error
}

type service struct {
	repo document.Repository
}

func NewService(repo document.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, d *document.Document, actor string) (*document.Document, error) {
	doc := d.Clone()
	doc.State = document.DocumentStateActive
	doc.CreatedBy = actor
	doc.UpdatedBy = actor

	if err := document.ValidateDocument(doc); err != nil {
		return nil, err
	}

	n, err := s.repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	doc.Code = fmt.Sprintf("DOC-%05d", n)
	doc.ID = core.RowID("document", doc.Code)

	now := core.NowRFC3339()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *service) Get(ctx context.Context, id string) (*document.Document, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, patch *document.Document, actor string) (*document.Document, error) {
	cur, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.State != document.DocumentStateActive {
		return nil, document.ErrInvalid
	}

	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Type != "" {
		cur.Type = patch.Type
	}
	if patch.Folder != "" {
		cur.Folder = patch.Folder
	}
	if patch.URL != "" {
		cur.URL = patch.URL
	}
	if patch.Description != "" {
		cur.Description = patch.Description
	}
	if patch.Tags != nil {
		cur.Tags = patch.Tags
	}

	if err := document.ValidateDocument(cur); err != nil {
		return nil, err
	}

	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) List(ctx context.Context, owner string, docType string, state string) ([]*document.Document, error) {
	var dt document.DocumentType
	if docType != "" {
		dt = document.DocumentType(docType)
	}
	var ds document.DocumentState
	if state != "" {
		ds = document.DocumentState(state)
	}
	return s.repo.List(ctx, owner, dt, ds)
}

func (s *service) Archive(ctx context.Context, id, actor string) (*document.Document, error) {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc.State != document.DocumentStateActive {
		return nil, document.ErrInvalid
	}
	doc.State = document.DocumentStateArchived
	doc.ArchivedBy = actor
	doc.ArchivedAt = core.NowRFC3339()
	doc.UpdatedBy = actor
	doc.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *service) Delete(ctx context.Context, id, actor string) error {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if doc.RefCount > 0 {
		return document.ErrInvalid
	}
	doc.State = document.DocumentStateDeleted
	doc.DeletedBy = actor
	doc.DeletedAt = core.NowRFC3339()
	doc.UpdatedBy = actor
	doc.UpdatedAt = core.NowRFC3339()

	return s.repo.Update(ctx, doc)
}
