package budget

import (
	"context"
	"testing"

	"goGL/internal/domain/budget"
)

func TestCreatePlan_Success(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{}}
	svc := NewService(repo)

	input := &budget.BudgetPlan{
		Name:       "Ngân sách 2026",
		FiscalYear: 2026,
		Department: "Kế toán",
		Items: []budget.BudgetItem{
			{CategoryCode: "salary", Planned: 100000000},
			{CategoryCode: "office", Planned: 50000000},
		},
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != budget.BudgetStateDraft {
		t.Errorf("state = %q, want draft", result.State)
	}
	if result.TotalPlanned != 150000000 {
		t.Errorf("total_planned = %d, want 150000000", result.TotalPlanned)
	}
}

func TestCreatePlan_EmptyName(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{}}
	svc := NewService(repo)

	input := &budget.BudgetPlan{
		FiscalYear: 2026,
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreatePlan_InvalidYear(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{}}
	svc := NewService(repo)

	input := &budget.BudgetPlan{
		Name:       "Test",
		FiscalYear: 2019,
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for invalid fiscal year")
	}
}

func TestCreatePlan_NegativePlanned(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{}}
	svc := NewService(repo)

	input := &budget.BudgetPlan{
		Name:       "Test",
		FiscalYear: 2026,
		Items: []budget.BudgetItem{
			{CategoryCode: "salary", Planned: -100},
		},
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for negative planned amount")
	}
}

func TestGetPlan_Success(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", Name: "Test Plan"},
	}}
	svc := NewService(repo)

	result, err := svc.Get(context.Background(), "plan-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test Plan" {
		t.Errorf("name = %q, want Test Plan", result.Name)
	}
}

func TestGetPlan_NotFound(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{}}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestUpdatePlan_Draft(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", Name: "Old", State: budget.BudgetStateDraft, FiscalYear: 2026},
	}}
	svc := NewService(repo)

	patch := &budget.BudgetPlan{Name: "New Name"}
	result, err := svc.Update(context.Background(), "plan-1", patch, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "New Name" {
		t.Errorf("name = %q, want New Name", result.Name)
	}
}

func TestUpdatePlan_Locked(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateLocked},
	}}
	svc := NewService(repo)

	patch := &budget.BudgetPlan{Name: "New Name"}
	_, err := svc.Update(context.Background(), "plan-1", patch, "admin")
	if err == nil {
		t.Error("expected error for locked plan")
	}
}

func TestApprovePlan_Success(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateDraft},
	}}
	svc := NewService(repo)

	result, err := svc.Approve(context.Background(), "plan-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != budget.BudgetStateApproved {
		t.Errorf("state = %q, want approved", result.State)
	}
	if result.ApprovedBy != "admin" {
		t.Errorf("approved_by = %q, want admin", result.ApprovedBy)
	}
}

func TestApprovePlan_WrongState(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateLocked},
	}}
	svc := NewService(repo)

	_, err := svc.Approve(context.Background(), "plan-1", "admin")
	if err == nil {
		t.Error("expected error for locked plan")
	}
}

func TestLockPlan_Success(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateApproved},
	}}
	svc := NewService(repo)

	result, err := svc.Lock(context.Background(), "plan-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != budget.BudgetStateLocked {
		t.Errorf("state = %q, want locked", result.State)
	}
}

func TestLockPlan_WrongState(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateDraft},
	}}
	svc := NewService(repo)

	_, err := svc.Lock(context.Background(), "plan-1", "admin")
	if err == nil {
		t.Error("expected error for draft plan")
	}
}

func TestDeletePlan_Draft(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateDraft},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "plan-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePlan_Approved(t *testing.T) {
	repo := &mockRepo{plans: map[string]*budget.BudgetPlan{
		"plan-1": {ID: "plan-1", State: budget.BudgetStateApproved},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "plan-1")
	if err == nil {
		t.Error("expected error for approved plan")
	}
}

func TestBudgetState_IsValid(t *testing.T) {
	tests := []struct {
		s    budget.BudgetState
		want bool
	}{
		{budget.BudgetStateDraft, true},
		{budget.BudgetStateApproved, true},
		{budget.BudgetStateLocked, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("BudgetState(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestRecalculate(t *testing.T) {
	p := &budget.BudgetPlan{
		Items: []budget.BudgetItem{
			{Planned: 100, Actual: 80},
			{Planned: 200, Actual: 150},
		},
	}
	p.Recalculate()
	if p.TotalPlanned != 300 {
		t.Errorf("total_planned = %d, want 300", p.TotalPlanned)
	}
	if p.TotalActual != 230 {
		t.Errorf("total_actual = %d, want 230", p.TotalActual)
	}
}

// mockRepo is an in-memory repository for testing.
type mockRepo struct {
	plans map[string]*budget.BudgetPlan
	seq   int64
}

func (m *mockRepo) Create(_ context.Context, p *budget.BudgetPlan) error {
	m.plans[p.ID] = p
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id string) (*budget.BudgetPlan, error) {
	if p, ok := m.plans[id]; ok {
		return p, nil
	}
	return nil, budget.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, p *budget.BudgetPlan) error {
	m.plans[p.ID] = p
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	delete(m.plans, id)
	return nil
}

func (m *mockRepo) List(_ context.Context, fiscalYear int, department string) ([]*budget.BudgetPlan, error) {
	var out []*budget.BudgetPlan
	for _, p := range m.plans {
		if fiscalYear > 0 && p.FiscalYear != fiscalYear {
			continue
		}
		if department != "" && p.Department != department {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (m *mockRepo) NextCode(_ context.Context, fiscalYear int) (int64, error) {
	m.seq++
	return m.seq, nil
}
