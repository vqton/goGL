package masterdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// Kind enumerates the master-data catalogs owned by this module. The string
// values are stable on purpose: they appear in URLs, JSON payloads and
// persisted JSON documents.
type Kind string

const (
	KindAccount       Kind = "account"
	KindCustomer      Kind = "customer"
	KindSupplier      Kind = "supplier"
	KindItem          Kind = "item"
	KindUnit          Kind = "unit"
	KindWarehouse     Kind = "warehouse"
	KindDepartment    Kind = "department"
	KindEmployee      Kind = "employee"
	KindFixedAssetCat Kind = "fixed_asset_cat"
	KindBank          Kind = "bank"
	KindCurrency      Kind = "currency"
	KindTaxRate       Kind = "tax_rate"
	KindFund          Kind = "fund"
	KindCostObject    Kind = "cost_object"
	KindReason        Kind = "reason"
	KindCustomerGroup Kind = "customer_group"
	KindSupplierGroup Kind = "supplier_group"
	KindItemGroup     Kind = "item_group"
)

// Kinds is the canonical, ordered list of supported catalogs.
var Kinds = []Kind{
	KindAccount, KindCustomer, KindSupplier, KindItem, KindUnit, KindWarehouse,
	KindDepartment, KindEmployee, KindFixedAssetCat, KindBank, KindCurrency,
	KindTaxRate, KindFund, KindCostObject, KindReason,
	KindCustomerGroup, KindSupplierGroup, KindItemGroup,
}

var kindLabels = map[Kind]string{
	KindAccount:       "Tài khoản kế toán",
	KindCustomer:      "Khách hàng",
	KindSupplier:      "Nhà cung cấp",
	KindItem:          "Hàng hóa, vật tư, dịch vụ",
	KindUnit:          "Đơn vị tính",
	KindWarehouse:     "Kho",
	KindDepartment:    "Phòng ban",
	KindEmployee:      "Nhân viên",
	KindFixedAssetCat: "Loại tài sản cố định",
	KindBank:          "Ngân hàng",
	KindCurrency:      "Loại tiền tệ",
	KindTaxRate:       "Thuế suất",
	KindFund:          "Quỹ",
	KindCostObject:    "Đối tượng tập hợp chi phí",
	KindReason:        "Lý do",
	KindCustomerGroup: "Nhóm khách hàng",
	KindSupplierGroup: "Nhóm nhà cung cấp",
	KindItemGroup:     "Nhóm hàng hóa",
}

var kindPrefixes = map[Kind]string{
	KindAccount:       "",
	KindCustomer:      "KH",
	KindSupplier:      "NCC",
	KindItem:          "VT",
	KindUnit:          "DVT",
	KindWarehouse:     "KHO",
	KindDepartment:    "BP",
	KindEmployee:      "NV",
	KindFixedAssetCat: "TSCD",
	KindBank:          "NH",
	KindCurrency:      "NT",
	KindTaxRate:       "TS",
	KindFund:          "QUY",
	KindCostObject:    "CT",
	KindReason:        "LD",
	KindCustomerGroup: "NKH",
	KindSupplierGroup: "NNCC",
	KindItemGroup:     "NVL",
}

// IsKind reports whether k is a known catalog kind.
func (k Kind) IsKind() bool {
	for _, kk := range Kinds {
		if k == kk {
			return true
		}
	}
	return false
}

// Label returns the Vietnamese display label for the kind.
func (k Kind) Label() string {
	if s, ok := kindLabels[k]; ok {
		return s
	}
	return string(k)
}

// Prefix returns the prefix used for auto-generated codes (e.g. KH-00001).
func (k Kind) Prefix() string {
	return kindPrefixes[k]
}

// AutoCode reports whether the kind supports auto-generated codes. Accounts
// are typed manually (numeric chart), so they are excluded.
func (k Kind) AutoCode() bool {
	return k != KindAccount
}

// CodePattern validates manually supplied codes. Accounts use a strict
// numeric chart rule; the rest accept letters/digits/._- (max 31 chars).
var (
	accountCodeRe = regexp.MustCompile(`^[0-9]{1,10}$`)
	manualCodeRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,30}$`)
	taxCodeRe     = regexp.MustCompile(`^[0-9]{10}$|^[0-9]{10}-[0-9]{3}$`)
	dateRe        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// AccountType classifies accounts for reporting (TT 99/2025 / TT 200/2014).
type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeRevenue   AccountType = "revenue"
	AccountTypeExpense   AccountType = "expense"
	AccountTypeResult    AccountType = "result"
)

// State is the lifecycle state of a master-data record.
type State string

const (
	StateActive   State = "active"
	StateInactive State = "inactive"
)

// Record is the generic master-data row persisted as a JSON document in the
// md_records table. Extra holds kind-specific attributes (tax_code, rate,
// unit_code, ...) as strings so the schema stays uniform.
type Record struct {
	ID               string            `json:"id"`
	Kind             Kind              `json:"kind"`
	Code             string            `json:"code"`
	Name             string            `json:"name"`
	NameEN           string            `json:"name_en,omitempty"`
	GroupCode        string            `json:"group_code,omitempty"`
	State            State             `json:"state"`
	Level            int               `json:"level,omitempty"`
	AccountType      AccountType       `json:"account_type,omitempty"`
	AllowPost        bool              `json:"allow_post,omitempty"`
	ValidFrom        string            `json:"valid_from,omitempty"`
	ValidTo          string            `json:"valid_to,omitempty"`
	RefCount         int64             `json:"ref_count,omitempty"`
	Extra            map[string]string `json:"extra,omitempty"`
	CreatedBy        string            `json:"created_by,omitempty"`
	CreatedAt        string            `json:"created_at"`
	UpdatedBy        string            `json:"updated_by,omitempty"`
	UpdatedAt        string            `json:"updated_at"`
	DeactivatedBy    string            `json:"deactivated_by,omitempty"`
	DeactivatedAt    string            `json:"deactivated_at,omitempty"`
	DeactivateReason string            `json:"deactivate_reason,omitempty"`
}

// Clone returns a deep copy safe for callers to mutate.
func (r *Record) Clone() *Record {
	cp := *r
	if r.Extra != nil {
		cp.Extra = make(map[string]string, len(r.Extra))
		for k, v := range r.Extra {
			cp.Extra[k] = v
		}
	}
	return &cp
}

// RecordID derives the deterministic row id from kind+code so re-writing the
// same record is an upsert, matching the (id, data) document table shape.
func RecordID(kind Kind, code string) string {
	sum := sha256.Sum256([]byte(string(kind) + "\x00" + code))
	return hex.EncodeToString(sum[:])
}

// Repository persists master-data records as JSON documents. List returns all
// rows for a kind, ordered by code; filtering and pagination live in the
// service (datasets are small in a single-tenant SQLite deployment).
type Repository interface {
	Upsert(ctx context.Context, r *Record) error
	Get(ctx context.Context, id string) (*Record, error)
	GetByCode(ctx context.Context, kind Kind, code string) (*Record, error)
	List(ctx context.Context, kind Kind) ([]*Record, error)
	Delete(ctx context.Context, id string) error
	NextCode(ctx context.Context, kind Kind) (int64, error)
	GetRegime(ctx context.Context) (string, error)
	SetRegime(ctx context.Context, regime, actor string) error

	// BudgetRecord operations
	UpsertBudget(ctx context.Context, b *BudgetRecord) error
	GetBudget(ctx context.Context, departmentCode string, fiscalYear int) (*BudgetRecord, error)
	ListBudgets(ctx context.Context, fiscalYear int) ([]*BudgetRecord, error)
	DeleteBudget(ctx context.Context, id string) error
}

// Registry is the cross-module seam other features use to resolve a master
// record (e.g. a customer code) into its stable identity. No consumer exists
// yet — the consumers (invoice, purchase, ...) are stubs — but the seam is
// closed so wiring later is additive.
type Registry struct {
	Lookup func(ctx context.Context, kind Kind, code string) (*Record, error)
}

// MergeResult reports a completed or dry-run merge.
type MergeResult struct {
	Keep     string           `json:"keep"`
	Merged   []string         `json:"merged"`
	Impacted map[string]int64 `json:"impacted"`
	DryRun   bool             `json:"dry_run"`
}

// RowError reports a single CSV row import problem.
type RowError struct {
	Row     int    `json:"row"`
	Column  string `json:"column,omitempty"`
	Message string `json:"message"`
}

// ImportResult summarizes a CSV import (create/update counts + errors).
type ImportResult struct {
	Total   int        `json:"total"`
	Created int        `json:"created"`
	Updated int        `json:"updated"`
	Errors  []RowError `json:"errors,omitempty"`
	DryRun  bool       `json:"dry_run"`
}

// NormalizeTaxCode strips whitespace so duplicate detection is forgiving.
func NormalizeTaxCode(v string) string {
	return strings.ReplaceAll(strings.TrimSpace(v), " ", "")
}

// ValidTaxCode reports whether v looks like a VN tax code: 10 digits, or a
// 13-char dependent-unit code (10 digits + "-" + 3 digits). NĐ 254/2026.
func ValidTaxCode(v string) bool {
	return taxCodeRe.MatchString(NormalizeTaxCode(v))
}

// ValidDate reports whether v is an RFC3339 date (YYYY-MM-DD) or empty.
func ValidDate(v string) bool {
	return v == "" || dateRe.MatchString(v)
}

func (k Kind) String() string { return string(k) }

// DepartmentType classifies departments for organizational structure (Circular 99/2025).
type DepartmentType string

const (
	// DepartmentTypeExecutive represents executive management (Ban Giám đốc).
	DepartmentTypeExecutive DepartmentType = "executive"
	// DepartmentTypeOperational represents operational departments (Phòng vận hành).
	DepartmentTypeOperational DepartmentType = "operational"
	// DepartmentTypeSupport represents support departments (Phòng hỗ trợ).
	DepartmentTypeSupport DepartmentType = "support"
)

// IsValid reports whether dt is a known department type.
func (dt DepartmentType) IsValid() bool {
	switch dt {
	case DepartmentTypeExecutive, DepartmentTypeOperational, DepartmentTypeSupport:
		return true
	default:
		return false
	}
}

// String returns the string representation of the department type.
func (dt DepartmentType) String() string { return string(dt) }

// DepartmentNode represents a department in the tree hierarchy.
type DepartmentNode struct {
	*Record
	Children []*DepartmentNode
}

// String returns a string representation of the department node for debugging.
func (n *DepartmentNode) String() string {
	if n == nil {
		return "<nil>"
	}
	return n.Code + " " + n.Name
}

// BudgetRecord holds budget data for a department per fiscal year.
type BudgetRecord struct {
	ID             string `json:"id"`
	DepartmentCode string `json:"department_code"`
	FiscalYear     int    `json:"fiscal_year"`
	Amount         int64  `json:"amount"`
	Notes          string `json:"notes,omitempty"`
	CreatedBy      string `json:"created_by,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedBy      string `json:"updated_by,omitempty"`
	UpdatedAt      string `json:"updated_at"`
}

// BuildDepartmentTree constructs a tree structure from a flat list of departments.
func BuildDepartmentTree(departments []*Record) []*DepartmentNode {
	nodeMap := make(map[string]*DepartmentNode)
	var roots []*DepartmentNode

	// Create nodes
	for _, dept := range departments {
		nodeMap[dept.Code] = &DepartmentNode{Record: dept}
	}

	// Build tree
	for _, dept := range departments {
		node := nodeMap[dept.Code]
		if dept.GroupCode == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[dept.GroupCode]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}
