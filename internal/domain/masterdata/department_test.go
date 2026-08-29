package masterdata

import "testing"

func TestDepartmentType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		input    DepartmentType
		expected bool
	}{
		{"executive is valid", DepartmentTypeExecutive, true},
		{"operational is valid", DepartmentTypeOperational, true},
		{"support is valid", DepartmentTypeSupport, true},
		{"invalid type", DepartmentType("invalid"), false},
		{"empty type", DepartmentType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.IsValid(); got != tt.expected {
				t.Errorf("DepartmentType(%q).IsValid() = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDepartmentType_String(t *testing.T) {
	tests := []struct {
		input    DepartmentType
		expected string
	}{
		{DepartmentTypeExecutive, "executive"},
		{DepartmentTypeOperational, "operational"},
		{DepartmentTypeSupport, "support"},
	}

	for _, tt := range tests {
		if got := tt.input.String(); got != tt.expected {
			t.Errorf("DepartmentType(%q).String() = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDepartmentNode(t *testing.T) {
	root := &DepartmentNode{
		Record: &Record{Code: "BP-00001", Name: "Ban Giám đốc"},
		Children: []*DepartmentNode{
			{Record: &Record{Code: "BP-00002", Name: "Phòng Kế toán"}},
			{Record: &Record{Code: "BP-00003", Name: "Phòng Kinh doanh"}},
		},
	}

	if root.Code != "BP-00001" {
		t.Errorf("root.Code = %q, want BP-00001", root.Code)
	}
	if len(root.Children) != 2 {
		t.Errorf("root.Children count = %d, want 2", len(root.Children))
	}
	if root.Children[0].Code != "BP-00002" {
		t.Errorf("child[0].Code = %q, want BP-00002", root.Children[0].Code)
	}
}

func TestBuildDepartmentTree_SingleRoot(t *testing.T) {
	departments := []*Record{
		{Code: "BP-00001", Name: "Ban Giám đốc", GroupCode: ""},
		{Code: "BP-00002", Name: "Phòng Kế toán", GroupCode: "BP-00001"},
		{Code: "BP-00003", Name: "Phòng Kinh doanh", GroupCode: "BP-00001"},
	}

	tree := BuildDepartmentTree(departments)

	if len(tree) != 1 {
		t.Fatalf("root nodes = %d, want 1", len(tree))
	}
	if tree[0].Code != "BP-00001" {
		t.Errorf("root code = %q, want BP-00001", tree[0].Code)
	}
	if len(tree[0].Children) != 2 {
		t.Errorf("root children = %d, want 2", len(tree[0].Children))
	}
}

func TestBuildDepartmentTree_MultipleRoots(t *testing.T) {
	departments := []*Record{
		{Code: "BP-00001", Name: "Ban Giám đốc", GroupCode: ""},
		{Code: "BP-00002", Name: "Ban Giám đốc 2", GroupCode: ""},
	}

	tree := BuildDepartmentTree(departments)

	if len(tree) != 2 {
		t.Fatalf("root nodes = %d, want 2", len(tree))
	}
}

func TestBuildDepartmentTree_NestedHierarchy(t *testing.T) {
	departments := []*Record{
		{Code: "BP-00001", Name: "Ban Giám đốc", GroupCode: ""},
		{Code: "BP-00002", Name: "Phòng Kế toán", GroupCode: "BP-00001"},
		{Code: "BP-00003", Name: "Tổ Kế toán", GroupCode: "BP-00002"},
	}

	tree := BuildDepartmentTree(departments)

	if len(tree) != 1 {
		t.Fatalf("root nodes = %d, want 1", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(tree[0].Children))
	}
	if len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("grandchildren = %d, want 1", len(tree[0].Children[0].Children))
	}
}

func TestBuildDepartmentTree_Empty(t *testing.T) {
	departments := []*Record{}

	tree := BuildDepartmentTree(departments)

	if len(tree) != 0 {
		t.Errorf("root nodes = %d, want 0", len(tree))
	}
}
