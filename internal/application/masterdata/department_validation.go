package masterdata

import (
	"regexp"
	"strings"

	"goGL/internal/domain/masterdata"
)

// departmentCodeRe matches the department code format: BP-XXXXX
var departmentCodeRe = regexp.MustCompile(`^BP-[0-9]{5}$`)

// costCenterCodeRe matches the cost center code format: CC-XXX
var costCenterCodeRe = regexp.MustCompile(`^CC-[0-9]{3,}$`)

// validateDepartment validates department-specific fields.
// This implements BR-D001 to BR-D010 for the Department module.
func (s *service) validateDepartment(rec *masterdata.Record) error {
	// BR-D001: Code format (BP-XXXXX)
	if rec.Code != "" {
		if !departmentCodeRe.MatchString(rec.Code) {
			return &masterdata.ValidationError{
				Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Mã phòng ban phải đúng định dạng BP-XXXXX",
				MessageEn: "department code must be in format BP-XXXXX",
			}
		}
	}

	// BR-D002: Name required, max 200 chars
	if strings.TrimSpace(rec.Name) == "" {
		return &masterdata.ValidationError{
			Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Tên phòng ban là bắt buộc",
			MessageEn: "department name is required",
		}
	}
	if len(rec.Name) > 200 {
		return &masterdata.ValidationError{
			Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Tên phòng ban tối đa 200 ký tự",
			MessageEn: "department name max 200 characters",
		}
	}

	// BR-D009: Department type validation
	if dt := masterdata.DepartmentType(rec.Extra["department_type"]); dt != "" && !dt.IsValid() {
		return &masterdata.ValidationError{
			Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Loại phòng ban không hợp lệ",
			MessageEn: "invalid department type",
		}
	}

	// BR-D005: Cost center format validation
	if cc := rec.Extra["cost_center_code"]; cc != "" {
		if !costCenterCodeRe.MatchString(cc) {
			return &masterdata.ValidationError{
				Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Mã trung tâm chi phí phải đúng định dạng CC-XXX",
				MessageEn: "cost center code must be in format CC-XXX",
			}
		}
	}

	return nil
}
