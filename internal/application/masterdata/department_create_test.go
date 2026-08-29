package masterdata

import (
	"context"
	"fmt"
	"testing"

	"goGL/internal/domain/masterdata"
)

func TestCreateDepartment_Success(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
	}
	svc := NewService(repo)

	input := &masterdata.Record{
		Name: "Phòng Kế toán",
		Extra: map[string]string{
			"department_type":  "support",
			"cost_center_code": "CC-001",
		},
	}

	result, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != masterdata.StateActive {
		t.Errorf("state = %q, want active", result.State)
	}
	if result.CreatedBy != "admin" {
		t.Errorf("created_by = %q, want admin", result.CreatedBy)
	}
}

func TestCreateDepartment_InvalidCode(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
	}
	svc := NewService(repo)

	input := &masterdata.Record{
		Code: "INVALID-001",
		Name: "Phòng Kế toán",
		Extra: map[string]string{
			"department_type": "support",
		},
	}

	_, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")

	if err == nil {
		t.Error("expected error for invalid code")
	}
}

func TestCreateDepartment_EmptyName(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
	}
	svc := NewService(repo)

	input := &masterdata.Record{
		Name: "",
		Extra: map[string]string{
			"department_type": "support",
		},
	}

	_, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")

	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateDepartment_InvalidType(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
	}
	svc := NewService(repo)

	input := &masterdata.Record{
		Name: "Phòng Kế toán",
		Extra: map[string]string{
			"department_type": "invalid",
		},
	}

	_, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")

	if err == nil {
		t.Error("expected error for invalid department type")
	}
}

func TestCreateDepartment_DuplicateCode(t *testing.T) {
	// Pre-populate with a record that has the same code
	existingID := masterdata.RecordID(masterdata.KindDepartment, "BP-00001")
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			existingID: {
				ID:   existingID,
				Kind: masterdata.KindDepartment,
				Code: "BP-00001",
				Name: "Existing Department",
			},
		},
	}
	svc := NewService(repo)

	input := &masterdata.Record{
		Code: "BP-00001",
		Name: "Another Department",
		Extra: map[string]string{
			"department_type": "support",
		},
	}

	_, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")

	if err == nil {
		t.Error("expected error for duplicate code")
	}
}

func TestCreateDepartment_InvalidCostCenter(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
	}
	svc := NewService(repo)

	input := &masterdata.Record{
		Name: "Phòng Kế toán",
		Extra: map[string]string{
			"department_type":  "support",
			"cost_center_code": "INVALID",
		},
	}

	_, err := svc.Create(context.Background(), masterdata.KindDepartment, input, "admin")

	if err == nil {
		t.Error("expected error for invalid cost center format")
	}
}

// mockRepository is a simple in-memory repository for testing.
type mockRepository struct {
	records map[string]*masterdata.Record
	budgets map[string]*masterdata.BudgetRecord
}

func (m *mockRepository) Upsert(ctx context.Context, r *masterdata.Record) error {
	m.records[r.ID] = r
	return nil
}

func (m *mockRepository) Get(ctx context.Context, id string) (*masterdata.Record, error) {
	if r, ok := m.records[id]; ok {
		return r, nil
	}
	return nil, masterdata.ErrNotFound
}

func (m *mockRepository) GetByCode(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
	for _, r := range m.records {
		if r.Kind == kind && r.Code == code {
			return r, nil
		}
	}
	return nil, masterdata.ErrNotFound
}

func (m *mockRepository) List(ctx context.Context, kind masterdata.Kind) ([]*masterdata.Record, error) {
	var out []*masterdata.Record
	for _, r := range m.records {
		if r.Kind == kind {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	delete(m.records, id)
	return nil
}

func (m *mockRepository) NextCode(ctx context.Context, kind masterdata.Kind) (int64, error) {
	count := int64(0)
	for _, r := range m.records {
		if r.Kind == kind {
			count++
		}
	}
	return count + 1, nil
}

func (m *mockRepository) GetRegime(ctx context.Context) (string, error) {
	return "", nil
}

func (m *mockRepository) SetRegime(ctx context.Context, regime, actor string) error {
	return nil
}

func (m *mockRepository) UpsertBudget(ctx context.Context, b *masterdata.BudgetRecord) error {
	m.budgets[b.ID] = b
	return nil
}

func (m *mockRepository) GetBudget(ctx context.Context, departmentCode string, fiscalYear int) (*masterdata.BudgetRecord, error) {
	id := fmt.Sprintf("%s_%d", departmentCode, fiscalYear)
	if b, ok := m.budgets[id]; ok {
		return b, nil
	}
	return nil, masterdata.ErrNotFound
}

func (m *mockRepository) ListBudgets(ctx context.Context, fiscalYear int) ([]*masterdata.BudgetRecord, error) {
	var out []*masterdata.BudgetRecord
	for _, b := range m.budgets {
		if fiscalYear == 0 || b.FiscalYear == fiscalYear {
			out = append(out, b)
		}
	}
	return out, nil
}

func (m *mockRepository) DeleteBudget(ctx context.Context, id string) error {
	return nil
}
