package tax

import (
	"context"
	"testing"

	"goGL/internal/domain/tax"
)

func TestCreateDeclaration_Success(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{}}
	svc := NewService(repo)

	input := &tax.TaxDeclaration{
		TaxType: tax.TaxTypeVAT,
		Period:  "2026-Q1",
		Items: []tax.LineItem{
			{Description: "Sales", TaxRate: 1000, Taxable: 50000000, TaxAmount: 5000000},
			{Description: "Services", TaxRate: 1000, Taxable: 20000000, TaxAmount: 2000000},
		},
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != tax.StateDraft {
		t.Errorf("state = %q, want draft", result.State)
	}
	if result.TotalTax != 7000000 {
		t.Errorf("total_tax = %d, want 7000000", result.TotalTax)
	}
	if result.TotalTaxable != 70000000 {
		t.Errorf("total_taxable = %d, want 70000000", result.TotalTaxable)
	}
}

func TestCreateDeclaration_InvalidType(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{}}
	svc := NewService(repo)

	input := &tax.TaxDeclaration{
		TaxType: "invalid",
		Period:  "2026-Q1",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for invalid tax type")
	}
}

func TestCreateDeclaration_EmptyPeriod(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{}}
	svc := NewService(repo)

	input := &tax.TaxDeclaration{
		TaxType: tax.TaxTypeVAT,
		Period:  "",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty period")
	}
}

func TestCreateDeclaration_CodeFormat(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{}}
	svc := NewService(repo)

	result, _ := svc.Create(context.Background(), &tax.TaxDeclaration{
		TaxType: tax.TaxTypeVAT,
		Period:  "2026-Q1",
	}, "admin")
	if len(result.Code) != 12 || result.Code[:8] != "TAX-VAT-" {
		t.Errorf("code = %q, want TAX-VAT-XXXX format", result.Code)
	}
}

func TestGetDeclaration_Success(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", Period: "2026-Q1"},
	}}
	svc := NewService(repo)

	result, err := svc.Get(context.Background(), "tax-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Period != "2026-Q1" {
		t.Errorf("period = %q, want 2026-Q1", result.Period)
	}
}

func TestGetDeclaration_NotFound(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{}}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent declaration")
	}
}

func TestUpdateDeclaration_Draft(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", Period: "2026-Q1", State: tax.StateDraft, TaxType: tax.TaxTypeVAT},
	}}
	svc := NewService(repo)

	patch := &tax.TaxDeclaration{Period: "2026-Q2"}
	result, err := svc.Update(context.Background(), "tax-1", patch, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Period != "2026-Q2" {
		t.Errorf("period = %q, want 2026-Q2", result.Period)
	}
}

func TestUpdateDeclaration_Locked(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", State: tax.StateFiled},
	}}
	svc := NewService(repo)

	patch := &tax.TaxDeclaration{Period: "2026-Q2"}
	_, err := svc.Update(context.Background(), "tax-1", patch, "admin")
	if err == nil {
		t.Error("expected error for filed declaration")
	}
}

func TestFileDeclaration_Success(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", State: tax.StateDraft},
	}}
	svc := NewService(repo)

	result, err := svc.File(context.Background(), "tax-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != tax.StateFiled {
		t.Errorf("state = %q, want filed", result.State)
	}
	if result.FiledBy != "admin" {
		t.Errorf("filed_by = %q, want admin", result.FiledBy)
	}
}

func TestFileDeclaration_WrongState(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", State: tax.StateFiled},
	}}
	svc := NewService(repo)

	_, err := svc.File(context.Background(), "tax-1", "admin")
	if err == nil {
		t.Error("expected error for already filed declaration")
	}
}

func TestDeleteDeclaration_Draft(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", State: tax.StateDraft},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "tax-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDeclaration_Filed(t *testing.T) {
	repo := &mockRepo{decls: map[string]*tax.TaxDeclaration{
		"tax-1": {ID: "tax-1", State: tax.StateFiled},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "tax-1")
	if err == nil {
		t.Error("expected error for filed declaration")
	}
}

func TestTaxType_IsValid(t *testing.T) {
	tests := []struct {
		t    tax.TaxType
		want bool
	}{
		{tax.TaxTypeVAT, true},
		{tax.TaxTypeCIT, true},
		{tax.TaxTypePersonalIncome, true},
		{tax.TaxTypeImportDuty, true},
		{tax.TaxTypeExcise, true},
		{tax.TaxTypeOther, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.t.IsValid(); got != tt.want {
			t.Errorf("TaxType(%q).IsValid() = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestRecalculate(t *testing.T) {
	d := &tax.TaxDeclaration{
		Items: []tax.LineItem{
			{Taxable: 100000, TaxAmount: 10000},
			{Taxable: 200000, TaxAmount: 20000},
		},
	}
	d.Recalculate()
	if d.TotalTaxable != 300000 {
		t.Errorf("total_taxable = %d, want 300000", d.TotalTaxable)
	}
	if d.TotalTax != 30000 {
		t.Errorf("total_tax = %d, want 30000", d.TotalTax)
	}
}

// mockRepo is an in-memory repository for testing.
type mockRepo struct {
	decls map[string]*tax.TaxDeclaration
	seq   int64
}

func (m *mockRepo) Create(_ context.Context, d *tax.TaxDeclaration) error {
	m.decls[d.ID] = d
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id string) (*tax.TaxDeclaration, error) {
	if d, ok := m.decls[id]; ok {
		return d, nil
	}
	return nil, tax.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, d *tax.TaxDeclaration) error {
	m.decls[d.ID] = d
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	delete(m.decls, id)
	return nil
}

func (m *mockRepo) List(_ context.Context, taxType tax.TaxType, period string) ([]*tax.TaxDeclaration, error) {
	var out []*tax.TaxDeclaration
	for _, d := range m.decls {
		if taxType != "" && d.TaxType != taxType {
			continue
		}
		if period != "" && d.Period != period {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (m *mockRepo) NextCode(_ context.Context, taxType tax.TaxType) (int64, error) {
	m.seq++
	return m.seq, nil
}
