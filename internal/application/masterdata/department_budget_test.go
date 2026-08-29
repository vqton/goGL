package masterdata

import (
	"context"
	"testing"

	"goGL/internal/domain/masterdata"
)

func TestSetBudget_Success(t *testing.T) {
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			"department_BP-00001": {
				ID:   "department_BP-00001",
				Kind: masterdata.KindDepartment,
				Code: "BP-00001",
				Name: "Phòng Kế toán",
			},
		},
		budgets: make(map[string]*masterdata.BudgetRecord),
	}
	svc := NewService(repo)

	result, err := svc.SetBudget(context.Background(), "BP-00001", 2025, 1000000, "Test budget", "admin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Amount != 1000000 {
		t.Errorf("amount = %d, want 1000000", result.Amount)
	}
	if result.FiscalYear != 2025 {
		t.Errorf("fiscal_year = %d, want 2025", result.FiscalYear)
	}
	if result.DepartmentCode != "BP-00001" {
		t.Errorf("department_code = %q, want BP-00001", result.DepartmentCode)
	}
}

func TestSetBudget_NegativeAmount(t *testing.T) {
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			"department_BP-00001": {
				ID:   "department_BP-00001",
				Kind: masterdata.KindDepartment,
				Code: "BP-00001",
				Name: "Phòng Kế toán",
			},
		},
		budgets: make(map[string]*masterdata.BudgetRecord),
	}
	svc := NewService(repo)

	_, err := svc.SetBudget(context.Background(), "BP-00001", 2025, -1000, "Invalid", "admin")

	if err == nil {
		t.Error("expected error for negative amount")
	}
	if ve, ok := err.(*masterdata.ValidationError); ok {
		if ve.MessageEn != "budget amount must be non-negative" {
			t.Errorf("error message = %q, want budget amount must be non-negative", ve.MessageEn)
		}
	}
}

func TestSetBudget_DepartmentNotFound(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
		budgets: make(map[string]*masterdata.BudgetRecord),
	}
	svc := NewService(repo)

	_, err := svc.SetBudget(context.Background(), "BP-99999", 2025, 1000000, "Test", "admin")

	if err == nil {
		t.Error("expected error for non-existent department")
	}
}

func TestGetBudget_Success(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
		budgets: map[string]*masterdata.BudgetRecord{
			"BP-00001_2025": {
				ID:             "BP-00001_2025",
				DepartmentCode: "BP-00001",
				FiscalYear:     2025,
				Amount:         1000000,
			},
		},
	}
	svc := NewService(repo)

	result, err := svc.GetBudget(context.Background(), "BP-00001", 2025)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Amount != 1000000 {
		t.Errorf("amount = %d, want 1000000", result.Amount)
	}
}

func TestGetBudget_NotFound(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
		budgets: make(map[string]*masterdata.BudgetRecord),
	}
	svc := NewService(repo)

	_, err := svc.GetBudget(context.Background(), "BP-00001", 2025)

	if err == nil {
		t.Error("expected error for non-existent budget")
	}
}

func TestListBudgets_Success(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
		budgets: map[string]*masterdata.BudgetRecord{
			"BP-00001_2025": {
				ID:             "BP-00001_2025",
				DepartmentCode: "BP-00001",
				FiscalYear:     2025,
				Amount:         1000000,
			},
			"BP-00002_2025": {
				ID:             "BP-00002_2025",
				DepartmentCode: "BP-00002",
				FiscalYear:     2025,
				Amount:         2000000,
			},
		},
	}
	svc := NewService(repo)

	result, err := svc.ListBudgets(context.Background(), 2025)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("count = %d, want 2", len(result))
	}
}
