package masterdata

import (
	"testing"

	"goGL/internal/domain/masterdata"
)

func TestValidateDepartment_CodeFormat(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
		errMsg  string
	}{
		{"valid code", "BP-00001", false, ""},
		{"valid code long", "BP-12345", false, ""},
		{"invalid prefix", "DEP-00001", true, "department code must be in format BP-XXXXX"},
		{"invalid format short", "BP-000", true, "department code must be in format BP-XXXXX"},
		{"empty code", "", false, ""}, // Auto-generated
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service{}
			rec := &masterdata.Record{
				Kind: masterdata.KindDepartment,
				Code: tt.code,
				Name: "Test Department",
				Extra: map[string]string{
					"department_type": "support",
				},
			}
			err := svc.validateDepartment(rec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDepartment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if ve, ok := err.(*masterdata.ValidationError); ok {
					if ve.MessageEn != tt.errMsg {
						t.Errorf("error message = %q, want %q", ve.MessageEn, tt.errMsg)
					}
				}
			}
		})
	}
}

func TestValidateDepartment_NameRequired(t *testing.T) {
	svc := &service{}
	rec := &masterdata.Record{
		Kind: masterdata.KindDepartment,
		Code: "BP-00001",
		Name: "",
		Extra: map[string]string{
			"department_type": "support",
		},
	}

	err := svc.validateDepartment(rec)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidateDepartment_NameMaxLength(t *testing.T) {
	svc := &service{}
	rec := &masterdata.Record{
		Kind: masterdata.KindDepartment,
		Code: "BP-00001",
		Name: string(make([]byte, 201)), // 201 chars
		Extra: map[string]string{
			"department_type": "support",
		},
	}

	err := svc.validateDepartment(rec)
	if err == nil {
		t.Error("expected error for name > 200 chars")
	}
}

func TestValidateDepartment_TypeInvalid(t *testing.T) {
	svc := &service{}
	rec := &masterdata.Record{
		Kind: masterdata.KindDepartment,
		Code: "BP-00001",
		Name: "Test Department",
		Extra: map[string]string{
			"department_type": "invalid",
		},
	}

	err := svc.validateDepartment(rec)
	if err == nil {
		t.Error("expected error for invalid department type")
	}
}

func TestValidateDepartment_CostCenterFormat(t *testing.T) {
	tests := []struct {
		name       string
		costCenter string
		wantErr    bool
	}{
		{"valid cost center", "CC-001", false},
		{"empty cost center", "", false},
		{"invalid format", "INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service{}
			rec := &masterdata.Record{
				Kind: masterdata.KindDepartment,
				Code: "BP-00001",
				Name: "Test Department",
				Extra: map[string]string{
					"department_type":  "support",
					"cost_center_code": tt.costCenter,
				},
			}

			err := svc.validateDepartment(rec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDepartment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
