package contract

import (
	"context"
	"testing"

	"goGL/internal/domain/contract"
)

func TestCreateContract_Success(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	input := &contract.Contract{
		Name:      "Service Agreement",
		Type:      contract.TypeService,
		PartyName: "ABC Corp",
		StartDate: "2026-01-01",
		EndDate:   "2026-12-31",
		Value:     100000000,
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != contract.StateDraft {
		t.Errorf("state = %q, want draft", result.State)
	}
}

func TestCreateContract_EmptyName(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	input := &contract.Contract{
		Type:      contract.TypeService,
		PartyName: "ABC",
		StartDate: "2026-01-01",
		EndDate:   "2026-12-31",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateContract_EndBeforeStart(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	input := &contract.Contract{
		Name:      "Test",
		Type:      contract.TypeService,
		PartyName: "ABC",
		StartDate: "2026-12-31",
		EndDate:   "2026-01-01",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for end date before start date")
	}
}

func TestCreateContract_CodeFormat(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	result, _ := svc.Create(context.Background(), &contract.Contract{
		Name:      "Test",
		Type:      contract.TypeService,
		PartyName: "ABC",
		StartDate: "2026-01-01",
		EndDate:   "2026-12-31",
	}, "admin")
	if len(result.Code) != 9 || result.Code[:4] != "CTR-" {
		t.Errorf("code = %q, want CTR-XXXXX format", result.Code)
	}
}

func TestGetContract_Success(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", Name: "Test"},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	result, err := svc.Get(context.Background(), "ctr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test" {
		t.Errorf("name = %q, want Test", result.Name)
	}
}

func TestGetContract_NotFound(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent contract")
	}
}

func TestUpdateContract_Draft(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", Name: "Old", State: contract.StateDraft, StartDate: "2026-01-01", EndDate: "2026-12-31", PartyName: "ABC", Type: contract.TypeService},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	patch := &contract.Contract{Name: "New Name"}
	result, err := svc.Update(context.Background(), "ctr-1", patch, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "New Name" {
		t.Errorf("name = %q, want New Name", result.Name)
	}
}

func TestUpdateContract_Active(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateActive},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	patch := &contract.Contract{Name: "New Name"}
	_, err := svc.Update(context.Background(), "ctr-1", patch, "admin")
	if err == nil {
		t.Error("expected error for active contract")
	}
}

func TestActivateContract_Success(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateDraft, StartDate: "2026-01-01", EndDate: "2026-12-31"},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	result, err := svc.Activate(context.Background(), "ctr-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != contract.StateActive {
		t.Errorf("state = %q, want active", result.State)
	}
}

func TestActivateContract_WrongState(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateActive},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	_, err := svc.Activate(context.Background(), "ctr-1", "admin")
	if err == nil {
		t.Error("expected error for active contract")
	}
}

func TestTerminateContract_Success(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateActive},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	result, err := svc.Terminate(context.Background(), "ctr-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != contract.StateTerminated {
		t.Errorf("state = %q, want terminated", result.State)
	}
}

func TestTerminateContract_WrongState(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateDraft},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	_, err := svc.Terminate(context.Background(), "ctr-1", "admin")
	if err == nil {
		t.Error("expected error for draft contract")
	}
}

func TestDeleteContract_Draft(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateDraft},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "ctr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContract_Active(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{
			"ctr-1": {ID: "ctr-1", State: contract.StateActive},
		},
		loans: map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "ctr-1")
	if err == nil {
		t.Error("expected error for active contract")
	}
}

func TestCreateLoan_Success(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	input := &contract.LoanAgreement{
		Principal:        500000000,
		InterestRateBP:   850,
		TermMonths:       36,
		MonthlyPayment:   15750000,
		DisbursementDate: "2026-01-01",
	}

	result, err := svc.CreateLoan(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
}

func TestGetLoan_Success(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans: map[string]*contract.LoanAgreement{
			"loan-1": {ID: "loan-1", Principal: 100000},
		},
	}
	svc := NewService(repo)

	result, err := svc.GetLoan(context.Background(), "loan-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Principal != 100000 {
		t.Errorf("principal = %d, want 100000", result.Principal)
	}
}

func TestGetLoan_NotFound(t *testing.T) {
	repo := &mockRepo{
		contracts: map[string]*contract.Contract{},
		loans:     map[string]*contract.LoanAgreement{},
	}
	svc := NewService(repo)

	_, err := svc.GetLoan(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent loan")
	}
}

// mockRepo is an in-memory repository for testing.
type mockRepo struct {
	contracts map[string]*contract.Contract
	loans     map[string]*contract.LoanAgreement
	seq       int64
}

func (m *mockRepo) Create(_ context.Context, c *contract.Contract) error {
	m.contracts[c.ID] = c
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id string) (*contract.Contract, error) {
	if c, ok := m.contracts[id]; ok {
		return c, nil
	}
	return nil, contract.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, c *contract.Contract) error {
	m.contracts[c.ID] = c
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	delete(m.contracts, id)
	return nil
}

func (m *mockRepo) List(_ context.Context, ctype contract.ContractType, state contract.ContractState) ([]*contract.Contract, error) {
	var out []*contract.Contract
	for _, c := range m.contracts {
		if ctype != "" && c.Type != ctype {
			continue
		}
		if state != "" && c.State != state {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (m *mockRepo) NextCode(_ context.Context) (int64, error) {
	m.seq++
	return m.seq, nil
}

func (m *mockRepo) CreateLoan(_ context.Context, l *contract.LoanAgreement) error {
	m.loans[l.ID] = l
	return nil
}

func (m *mockRepo) FindLoanByID(_ context.Context, id string) (*contract.LoanAgreement, error) {
	if l, ok := m.loans[id]; ok {
		return l, nil
	}
	return nil, contract.ErrNotFound
}

func (m *mockRepo) UpdateLoan(_ context.Context, l *contract.LoanAgreement) error {
	m.loans[l.ID] = l
	return nil
}
