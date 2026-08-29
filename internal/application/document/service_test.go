package document

import (
	"context"
	"testing"

	"goGL/internal/domain/document"
)

func TestCreateDocument_Success(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{}}
	svc := NewService(repo)

	input := &document.Document{
		Name:  "Hóa đơn GTGT",
		Type:  document.DocumentTypeInvoice,
		Owner: "user-001",
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != document.DocumentStateActive {
		t.Errorf("state = %q, want active", result.State)
	}
	if result.CreatedBy != "admin" {
		t.Errorf("created_by = %q, want admin", result.CreatedBy)
	}
	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateDocument_EmptyName(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{}}
	svc := NewService(repo)

	input := &document.Document{
		Name: "",
		Type: document.DocumentTypeInvoice,
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateDocument_InvalidType(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{}}
	svc := NewService(repo)

	input := &document.Document{
		Name: "Test Doc",
		Type: "invalid_type",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestCreateDocument_DefaultType(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{}}
	svc := NewService(repo)

	input := &document.Document{
		Name: "Test Doc",
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != document.DocumentTypeOther {
		t.Errorf("type = %q, want other", result.Type)
	}
}

func TestCreateDocument_CodeFormat(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{}}
	svc := NewService(repo)

	result, _ := svc.Create(context.Background(), &document.Document{Name: "Test"}, "admin")
	if len(result.Code) != 9 || result.Code[:4] != "DOC-" {
		t.Errorf("code = %q, want DOC-XXXXX format", result.Code)
	}
}

func TestGetDocument_Success(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", Name: "Test Doc"},
	}}
	svc := NewService(repo)

	result, err := svc.Get(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test Doc" {
		t.Errorf("name = %q, want Test Doc", result.Name)
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{}}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
}

func TestUpdateDocument_Success(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", Name: "Old Name", State: document.DocumentStateActive},
	}}
	svc := NewService(repo)

	patch := &document.Document{Name: "New Name"}
	result, err := svc.Update(context.Background(), "doc-1", patch, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "New Name" {
		t.Errorf("name = %q, want New Name", result.Name)
	}
	if result.UpdatedBy != "admin" {
		t.Errorf("updated_by = %q, want admin", result.UpdatedBy)
	}
}

func TestUpdateDocument_Archived(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", Name: "Archived", State: document.DocumentStateArchived},
	}}
	svc := NewService(repo)

	patch := &document.Document{Name: "New Name"}
	_, err := svc.Update(context.Background(), "doc-1", patch, "admin")
	if err == nil {
		t.Error("expected error for archived document")
	}
}

func TestListDocuments_FilterByOwner(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"d1": {ID: "d1", Owner: "user-1", State: document.DocumentStateActive},
		"d2": {ID: "d2", Owner: "user-2", State: document.DocumentStateActive},
		"d3": {ID: "d3", Owner: "user-1", State: document.DocumentStateActive},
	}}
	svc := NewService(repo)

	results, err := svc.List(context.Background(), "user-1", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestArchiveDocument_Success(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", Name: "Test", State: document.DocumentStateActive},
	}}
	svc := NewService(repo)

	result, err := svc.Archive(context.Background(), "doc-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != document.DocumentStateArchived {
		t.Errorf("state = %q, want archived", result.State)
	}
}

func TestArchiveDocument_AlreadyArchived(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", State: document.DocumentStateArchived},
	}}
	svc := NewService(repo)

	_, err := svc.Archive(context.Background(), "doc-1", "admin")
	if err == nil {
		t.Error("expected error for already archived document")
	}
}

func TestDeleteDocument_Success(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", Name: "Test", State: document.DocumentStateActive, RefCount: 0},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "doc-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := repo.docs["doc-1"]
	if doc.State != document.DocumentStateDeleted {
		t.Errorf("state = %q, want deleted", doc.State)
	}
}

func TestDeleteDocument_WithRefs(t *testing.T) {
	repo := &mockRepo{docs: map[string]*document.Document{
		"doc-1": {ID: "doc-1", State: document.DocumentStateActive, RefCount: 3},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "doc-1", "admin")
	if err == nil {
		t.Error("expected error for document with references")
	}
}

func TestDocumentType_IsValid(t *testing.T) {
	tests := []struct {
		typ  document.DocumentType
		want bool
	}{
		{document.DocumentTypeInvoice, true},
		{document.DocumentTypeContract, true},
		{document.DocumentTypeReceipt, true},
		{document.DocumentTypeReport, true},
		{document.DocumentTypeOther, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.typ.IsValid(); got != tt.want {
			t.Errorf("DocumentType(%q).IsValid() = %v, want %v", tt.typ, got, tt.want)
		}
	}
}

func TestValidateDocument_NameTooLong(t *testing.T) {
	d := &document.Document{
		Name: string(make([]byte, 501)),
		Type: document.DocumentTypeOther,
	}
	err := document.ValidateDocument(d)
	if err == nil {
		t.Error("expected error for name > 500 chars")
	}
}

// mockRepo is an in-memory repository for testing.
type mockRepo struct {
	docs map[string]*document.Document
	seq  int64
}

func (m *mockRepo) Create(_ context.Context, d *document.Document) error {
	m.docs[d.ID] = d
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id string) (*document.Document, error) {
	if d, ok := m.docs[id]; ok {
		return d, nil
	}
	return nil, document.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, d *document.Document) error {
	m.docs[d.ID] = d
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	delete(m.docs, id)
	return nil
}

func (m *mockRepo) List(_ context.Context, owner string, docType document.DocumentType, state document.DocumentState) ([]*document.Document, error) {
	var out []*document.Document
	for _, d := range m.docs {
		if owner != "" && d.Owner != owner {
			continue
		}
		if docType != "" && d.Type != docType {
			continue
		}
		if state != "" && d.State != state {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (m *mockRepo) NextCode(_ context.Context) (int64, error) {
	m.seq++
	return m.seq, nil
}
