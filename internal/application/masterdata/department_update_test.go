package masterdata

import (
	"context"
	"testing"

	"goGL/internal/domain/masterdata"
)

func TestUpdateDepartment_Success(t *testing.T) {
	existingID := masterdata.RecordID(masterdata.KindDepartment, "BP-00001")
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			existingID: {
				ID:    existingID,
				Kind:  masterdata.KindDepartment,
				Code:  "BP-00001",
				Name:  "Phòng Kế toán",
				State: masterdata.StateActive,
				Extra: map[string]string{"department_type": "support"},
			},
		},
	}
	svc := NewService(repo)

	patch := &masterdata.Record{
		Name: "Phòng Kế toán Tổng hợp",
	}

	result, err := svc.Update(context.Background(), masterdata.KindDepartment, "BP-00001", patch, "admin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Phòng Kế toán Tổng hợp" {
		t.Errorf("name = %q, want Phòng Kế toán Tổng hợp", result.Name)
	}
	if result.UpdatedBy != "admin" {
		t.Errorf("updated_by = %q, want admin", result.UpdatedBy)
	}
}

func TestUpdateDepartment_NotFound(t *testing.T) {
	repo := &mockRepository{
		records: make(map[string]*masterdata.Record),
	}
	svc := NewService(repo)

	patch := &masterdata.Record{
		Name: "Updated Name",
	}

	_, err := svc.Update(context.Background(), masterdata.KindDepartment, "BP-99999", patch, "admin")

	if err == nil {
		t.Error("expected error for non-existent department")
	}
}

func TestUpdateDepartment_InvalidName(t *testing.T) {
	existingID := masterdata.RecordID(masterdata.KindDepartment, "BP-00001")
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			existingID: {
				ID:    existingID,
				Kind:  masterdata.KindDepartment,
				Code:  "BP-00001",
				Name:  "Phòng Kế toán",
				State: masterdata.StateActive,
				Extra: map[string]string{"department_type": "support"},
			},
		},
	}
	svc := NewService(repo)

	patch := &masterdata.Record{
		Name: "   ", // Whitespace-only name should be invalid
	}

	_, err := svc.Update(context.Background(), masterdata.KindDepartment, "BP-00001", patch, "admin")

	if err == nil {
		t.Error("expected error for whitespace-only name")
	}
}

func TestUpdateDepartment_InvalidType(t *testing.T) {
	existingID := masterdata.RecordID(masterdata.KindDepartment, "BP-00001")
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			existingID: {
				ID:    existingID,
				Kind:  masterdata.KindDepartment,
				Code:  "BP-00001",
				Name:  "Phòng Kế toán",
				State: masterdata.StateActive,
				Extra: map[string]string{"department_type": "support"},
			},
		},
	}
	svc := NewService(repo)

	patch := &masterdata.Record{
		Extra: map[string]string{"department_type": "invalid"},
	}

	_, err := svc.Update(context.Background(), masterdata.KindDepartment, "BP-00001", patch, "admin")

	if err == nil {
		t.Error("expected error for invalid department type")
	}
}

func TestUpdateDepartment_CodeImmutable(t *testing.T) {
	existingID := masterdata.RecordID(masterdata.KindDepartment, "BP-00001")
	repo := &mockRepository{
		records: map[string]*masterdata.Record{
			existingID: {
				ID:    existingID,
				Kind:  masterdata.KindDepartment,
				Code:  "BP-00001",
				Name:  "Phòng Kế toán",
				State: masterdata.StateActive,
				Extra: map[string]string{"department_type": "support"},
			},
		},
	}
	svc := NewService(repo)

	patch := &masterdata.Record{
		Code: "BP-00002", // Try to change code
		Name: "Updated Name",
	}

	_, err := svc.Update(context.Background(), masterdata.KindDepartment, "BP-00001", patch, "admin")

	if err == nil {
		t.Error("expected error for code change attempt")
	}
}
