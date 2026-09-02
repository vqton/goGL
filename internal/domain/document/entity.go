package document

import (
	"context"
	"errors"
	"strings"

	"goGL/internal/domain/core"
)

var (
	ErrNotFound  = errors.New("document: not found")
	ErrDuplicate = errors.New("document: duplicate")
	ErrInvalid   = errors.New("document: invalid input")
	ErrForbidden = errors.New("document: forbidden")
)

type DocumentType string

const (
	DocumentTypeInvoice  DocumentType = "invoice"
	DocumentTypeContract DocumentType = "contract"
	DocumentTypeReceipt  DocumentType = "receipt"
	DocumentTypeReport   DocumentType = "report"
	DocumentTypeOther    DocumentType = "other"
)

func (dt DocumentType) IsValid() bool {
	switch dt {
	case DocumentTypeInvoice, DocumentTypeContract, DocumentTypeReceipt,
		DocumentTypeReport, DocumentTypeOther:
		return true
	default:
		return false
	}
}

type DocumentState string

const (
	DocumentStateActive   DocumentState = "active"
	DocumentStateArchived DocumentState = "archived"
	DocumentStateDeleted  DocumentState = "deleted"
)

type Document struct {
	ID          string        `json:"id"`
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Type        DocumentType  `json:"type"`
	Folder      string        `json:"folder,omitempty"`
	URL         string        `json:"url,omitempty"`
	Owner       string        `json:"owner"`
	State       DocumentState `json:"state"`
	Description string        `json:"description,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	RefCount    int64         `json:"ref_count,omitempty"`
	CreatedBy   string        `json:"created_by,omitempty"`
	CreatedAt   string        `json:"created_at"`
	UpdatedBy   string        `json:"updated_by,omitempty"`
	UpdatedAt   string        `json:"updated_at"`
	ArchivedBy  string        `json:"archived_by,omitempty"`
	ArchivedAt  string        `json:"archived_at,omitempty"`
	DeletedBy   string        `json:"deleted_by,omitempty"`
	DeletedAt   string        `json:"deleted_at,omitempty"`
}

func (d *Document) Clone() *Document {
	cp := *d
	if d.Tags != nil {
		cp.Tags = make([]string, len(d.Tags))
		copy(cp.Tags, d.Tags)
	}
	return &cp
}

func ValidateDocument(d *Document) error {
	if strings.TrimSpace(d.Name) == "" {
		return &core.ValidationError{Field: "name", Message: "name is required"}
	}
	if len(d.Name) > 500 {
		return &core.ValidationError{Field: "name", Message: "name max 500 chars"}
	}
	if d.Type == "" {
		d.Type = DocumentTypeOther
	}
	if !d.Type.IsValid() {
		return &core.ValidationError{Field: "type", Message: "invalid document type"}
	}
	if d.State == "" {
		d.State = DocumentStateActive
	}
	return nil
}

type Repository interface {
	Create(ctx context.Context, d *Document) error
	FindByID(ctx context.Context, id string) (*Document, error)
	Update(ctx context.Context, d *Document) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, owner string, docType DocumentType, state DocumentState) ([]*Document, error)
	NextCode(ctx context.Context) (int64, error)
}
