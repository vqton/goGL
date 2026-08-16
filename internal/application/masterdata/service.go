package masterdata

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"goGL/internal/domain/masterdata"
)

var (
	manualCodeRe  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,30}$`)
	accountCodeRe = regexp.MustCompile(`^[0-9]{1,10}$`)
)

// DefaultRegime is the active reporting regime until explicitly switched.
const DefaultRegime = "TT99-2025"

var supportedRegimes = map[string]bool{
	"TT99-2025":  true,
	"TT133-2016": true,
	"TT200-2014": true,
}

// Service is the master-data application service. It encodes the module's
// business rules (R1–R13 in the module spec).
type Service interface {
	Create(ctx context.Context, kind masterdata.Kind, in *masterdata.Record, actor string) (*masterdata.Record, error)
	Update(ctx context.Context, kind masterdata.Kind, code string, patch *masterdata.Record, actor string) (*masterdata.Record, error)
	Get(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error)
	List(ctx context.Context, kind masterdata.Kind, q, group, state string, page, size int) ([]*masterdata.Record, int, error)
	Deactivate(ctx context.Context, kind masterdata.Kind, code, reason, actor string) (*masterdata.Record, error)
	ForceDeactivate(ctx context.Context, kind masterdata.Kind, code, reason, actor string) (*masterdata.Record, error)
	Activate(ctx context.Context, kind masterdata.Kind, code, actor string) (*masterdata.Record, error)
	Delete(ctx context.Context, kind masterdata.Kind, code, actor string) error
	Merge(ctx context.Context, kind masterdata.Kind, keep string, dupes []string, actor string, dryRun bool) (*masterdata.MergeResult, error)
	ImportRows(ctx context.Context, kind masterdata.Kind, rows [][]string, actor string, dryRun bool) (*masterdata.ImportResult, error)
	Export(ctx context.Context, kind masterdata.Kind, q, group, state string) ([]*masterdata.Record, error)
	SeedAccounts(ctx context.Context, actor string) (int, error)
	SetRegime(ctx context.Context, regime, actor string) error
	GetRegime(ctx context.Context) (string, error)
	References(ctx context.Context, kind masterdata.Kind, code string) (int64, error)
	SetReferenceCount(ctx context.Context, kind masterdata.Kind, code string, n int64) error
	PermissionNames() []string
}

type service struct {
	repo masterdata.Repository
}

// NewService builds the master-data service over a repository.
func NewService(repo masterdata.Repository) Service {
	return &service{repo: repo}
}

func (s *service) PermissionNames() []string {
	return []string{
		"catalog.read",
		"catalog.write",
		"catalog.import",
		"catalog.merge",
		"catalog.seed",
		"catalog.regime",
	}
}

// allowedItemTypes mirrors the item type codes used in Vietnamese ERPs:
// 21 hàng hóa, 31 thành phẩm, 41 bán thành phẩm, 51 vật tư, 61 dịch vụ.
var allowedItemTypes = map[string]bool{"21": true, "31": true, "41": true, "51": true, "61": true}

// extraIdentityKeys are the identity attributes that make a customer/supplier
// legally distinct; at least one is required.
var extraIdentityKeys = []string{"tax_code", "id_number", "passport_no", "budget_unit_code"}

// refExtraKeys are Extra attributes across other kinds that can hold a record
// code reference (used by merge to re-point references).
var refExtraKeys = map[string]bool{
	"customer_code": true, "supplier_code": true, "employee_code": true,
	"unit_code": true, "warehouse_code": true, "department_code": true,
	"bank_code": true, "currency_code": true, "tax_rate_code": true,
	"item_code": true, "account": true,
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func accountTypeOf(code string) masterdata.AccountType {
	if code == "" {
		return ""
	}
	switch code[0] {
	case '1', '2':
		return masterdata.AccountTypeAsset
	case '3':
		return masterdata.AccountTypeLiability
	case '4':
		return masterdata.AccountTypeEquity
	case '5', '7':
		return masterdata.AccountTypeRevenue
	case '6', '8':
		return masterdata.AccountTypeExpense
	default:
		return masterdata.AccountTypeResult
	}
}

// ---------------------------------------------------------------------------
// Validation (R1–R10)
// ---------------------------------------------------------------------------

func (s *service) validate(ctx context.Context, rec *masterdata.Record) error {
	if !rec.Kind.IsKind() {
		return masterdata.ErrUnknownKind
	}
	if strings.TrimSpace(rec.Name) == "" {
		return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Tên là bắt buộc", MessageEn: "name is required"}
	}
	if rec.State == "" {
		rec.State = masterdata.StateActive
	}
	if rec.State != masterdata.StateActive && rec.State != masterdata.StateInactive {
		return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Trạng thái không hợp lệ", MessageEn: "invalid state"}
	}
	if rec.Code != "" {
		re := manualCodeRe
		if rec.Kind == masterdata.KindAccount {
			re = accountCodeRe
		}
		if !re.MatchString(rec.Code) {
			return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Mã không đúng định dạng", MessageEn: "invalid code format"}
		}
	}
	if !masterdata.ValidDate(rec.ValidFrom) || !masterdata.ValidDate(rec.ValidTo) {
		return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Ngày hiệu lực không đúng định dạng YYYY-MM-DD", MessageEn: "invalid valid_from/valid_to date"}
	}

	// Group membership (R3–R5): parent must exist, be active, same kind, and
	// the chain must not cycle.
	if rec.GroupCode != "" && rec.GroupCode != rec.Code {
		parent, err := s.repo.GetByCode(ctx, rec.Kind, rec.GroupCode)
		if errors.Is(err, masterdata.ErrNotFound) {
			return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Mã nhóm không tồn tại", MessageEn: "group not found"}
		}
		if err != nil {
			return err
		}
		if parent.State != masterdata.StateActive {
			return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Mã nhóm đang ngừng hoạt động", MessageEn: "group is inactive"}
		}
		if parent.ID == rec.ID {
			return masterdata.ErrCycle
		}
		if err := s.checkCycle(ctx, rec.Kind, rec.Code, rec.GroupCode, 0); err != nil {
			return err
		}
		if rec.Kind == masterdata.KindAccount {
			rec.Level = parent.Level + 1
		}
	}
	if rec.Kind == masterdata.KindAccount && rec.Level == 0 && rec.GroupCode == "" {
		rec.Level = 0
	}

	switch rec.Kind {
	case masterdata.KindAccount:
		if rec.AccountType == "" {
			rec.AccountType = accountTypeOf(rec.Code)
		}
		children, err := s.children(ctx, rec.Kind, rec.Code)
		if err != nil {
			return err
		}
		// Only leaves may be posted to (R9). Parent with children: never postable.
		if len(children) > 0 {
			rec.AllowPost = false
		}

	case masterdata.KindCustomer, masterdata.KindSupplier:
		if err := s.validateIdentity(ctx, rec); err != nil {
			return err
		}

	case masterdata.KindTaxRate:
		if rate := rec.Extra["rate"]; rate != "" {
			if _, err := parseRate(rate); err != nil {
				return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
					MessageVn: "Thuế suất không hợp lệ", MessageEn: "invalid tax rate"}
			}
		}
		if err := s.checkTaxOverlap(ctx, rec); err != nil {
			return err
		}

	case masterdata.KindItem:
		if it := rec.Extra["item_type"]; it != "" && !allowedItemTypes[it] {
			return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Loại hàng hóa không hợp lệ (21/31/41/51/61)",
				MessageEn: "invalid item_type"}
		}
	}
	return nil
}

func (s *service) checkCycle(ctx context.Context, kind masterdata.Kind, code, group string, depth int) error {
	if depth > 32 {
		return masterdata.ErrCycle
	}
	if group == code {
		return masterdata.ErrCycle
	}
	if group == "" {
		return nil
	}
	parent, err := s.repo.GetByCode(ctx, kind, group)
	if errors.Is(err, masterdata.ErrNotFound) {
		return nil // reported by the parent-exists check
	}
	if err != nil {
		return err
	}
	if parent.GroupCode == code {
		return masterdata.ErrCycle
	}
	return s.checkCycle(ctx, kind, code, parent.GroupCode, depth+1)
}

// validateIdentity enforces R6/R7: a customer/supplier must carry at least one
// legal identity attribute and a well-formed, unique tax code when present.
func (s *service) validateIdentity(ctx context.Context, rec *masterdata.Record) error {
	identityPresent := false
	for _, key := range extraIdentityKeys {
		if strings.TrimSpace(rec.Extra[key]) != "" {
			identityPresent = true
		}
	}
	if !identityPresent {
		return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
			MessageVn: "Thiếu mã số thuế hoặc CMND/CCCD/Hộ chiếu (NĐ 254/2026)",
			MessageEn: "tax code or national id is required"}
	}
	if tc := masterdata.NormalizeTaxCode(rec.Extra["tax_code"]); tc != "" {
		rec.Extra["tax_code"] = tc
		if !masterdata.ValidTaxCode(tc) {
			return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Mã số thuế không hợp lệ", MessageEn: "invalid tax code"}
		}
		all, err := s.repo.List(ctx, rec.Kind)
		if err != nil {
			return err
		}
		for _, other := range all {
			if other.ID == rec.ID {
				continue
			}
			if masterdata.NormalizeTaxCode(other.Extra["tax_code"]) == tc {
				return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
					MessageVn: "Mã số thuế đã tồn tại", MessageEn: "tax code already in use"}
			}
		}
	}
	return nil
}

func parseRate(v string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(strings.ReplaceAll(v, ",", "."), "%g", &f)
	return f, err
}

// checkTaxOverlap rejects two active tax-rate records with overlapping validity.
func (s *service) checkTaxOverlap(ctx context.Context, rec *masterdata.Record) error {
	if rec.State != masterdata.StateActive {
		return nil
	}
	all, err := s.repo.List(ctx, rec.Kind)
	if err != nil {
		return err
	}
	for _, other := range all {
		if other.ID == rec.ID || other.State != masterdata.StateActive {
			continue
		}
		if (rec.ValidFrom == "" || other.ValidTo == "" || rec.ValidFrom <= other.ValidTo) &&
			(other.ValidFrom == "" || rec.ValidTo == "" || rec.ValidTo >= other.ValidFrom) {
			return &masterdata.ValidationError{Kind: rec.Kind, Code: rec.Code,
				MessageVn: "Khoảng hiệu lực trùng với mã " + other.Code,
				MessageEn: "validity overlaps another active record"}
		}
	}
	return nil
}

// children lists direct children of a record (same kind, GroupCode == code).
func (s *service) children(ctx context.Context, kind masterdata.Kind, code string) ([]*masterdata.Record, error) {
	all, err := s.repo.List(ctx, kind)
	if err != nil {
		return nil, err
	}
	var out []*masterdata.Record
	for _, r := range all {
		if r.GroupCode == code {
			out = append(out, r)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Create / Update / Get / List
// ---------------------------------------------------------------------------

func (s *service) Create(ctx context.Context, kind masterdata.Kind, in *masterdata.Record, actor string) (*masterdata.Record, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	rec := in.Clone()
	rec.Kind = kind
	rec.State = masterdata.StateActive

	if rec.Code == "" {
		if !kind.AutoCode() {
			return nil, &masterdata.ValidationError{Kind: kind, Code: rec.Code,
				MessageVn: "Phải nhập mã tài khoản", MessageEn: "account code is required"}
		}
		n, err := s.repo.NextCode(ctx, kind)
		if err != nil {
			return nil, err
		}
		rec.Code = fmt.Sprintf("%s-%05d", kind.Prefix(), n)
	}
	if rec.ID == "" {
		rec.ID = masterdata.RecordID(kind, rec.Code)
	}

	if err := s.validate(ctx, rec); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByCode(ctx, kind, rec.Code); err == nil {
		return nil, fmt.Errorf("%w: %s/%s", masterdata.ErrDuplicate, kind, rec.Code)
	} else if !errors.Is(err, masterdata.ErrNotFound) {
		return nil, err
	}

	now := nowRFC3339()
	rec.CreatedAt, rec.UpdatedAt = now, now
	rec.CreatedBy, rec.UpdatedBy = actor, actor
	if err := s.repo.Upsert(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// fieldsLockedWhenReferenced may not change once RefCount > 0 (R2).
var fieldsLockedWhenReferenced = []string{
	"group_code", "account_type", "allow_post", "level", "valid_from", "valid_to",
	"tax_code", "id_number", "passport_no", "budget_unit_code",
	"unit_code", "bank_code", "currency_code", "tax_rate_code", "account",
}

func lockedFieldChanged(cur, merged *masterdata.Record) *string {
	if cur.GroupCode != merged.GroupCode {
		return strptr("group_code")
	}
	if cur.AccountType != merged.AccountType {
		return strptr("account_type")
	}
	if cur.AllowPost != merged.AllowPost {
		return strptr("allow_post")
	}
	if cur.ValidFrom != merged.ValidFrom || cur.ValidTo != merged.ValidTo {
		return strptr("valid_from/valid_to")
	}
	for _, key := range fieldsLockedWhenReferenced {
		if cur.Extra[key] != merged.Extra[key] {
			return strptr(key)
		}
	}
	return nil
}

func strptr(s string) *string { return &s }

func (s *service) Update(ctx context.Context, kind masterdata.Kind, code string, patch *masterdata.Record, actor string) (*masterdata.Record, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	cur, err := s.repo.GetByCode(ctx, kind, code)
	if err != nil {
		return nil, err
	}
	if patch.Code != "" && patch.Code != code {
		return nil, masterdata.ErrCodeImmutable
	}

	merged := cur.Clone()
	applyPatch(merged, patch)
	if cur.RefCount > 0 {
		if field := lockedFieldChanged(cur, merged); field != nil {
			return nil, &masterdata.ValidationError{Kind: kind, Code: code,
				MessageVn: "Bản ghi đang được tham chiếu, không sửa được " + *field,
				MessageEn: "field locked while referenced: " + *field}
		}
	}
	if err := s.validate(ctx, merged); err != nil {
		return nil, err
	}

	merged.UpdatedAt = nowRFC3339()
	merged.UpdatedBy = actor
	if err := s.repo.Upsert(ctx, merged); err != nil {
		return nil, err
	}
	return merged, nil
}

func applyPatch(cur, patch *masterdata.Record) {
	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.NameEN != "" {
		cur.NameEN = patch.NameEN
	}
	if patch.GroupCode != "" {
		cur.GroupCode = patch.GroupCode
	}
	if patch.AccountType != "" {
		cur.AccountType = patch.AccountType
	}
	if patch.ValidFrom != "" {
		cur.ValidFrom = patch.ValidFrom
	}
	if patch.ValidTo != "" {
		cur.ValidTo = patch.ValidTo
	}
	for k, v := range patch.Extra {
		if v == "" {
			delete(cur.Extra, k)
		} else {
			if cur.Extra == nil {
				cur.Extra = map[string]string{}
			}
			cur.Extra[k] = v
		}
	}
}

func (s *service) Get(ctx context.Context, kind masterdata.Kind, code string) (*masterdata.Record, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	return s.repo.GetByCode(ctx, kind, code)
}

func (s *service) Export(ctx context.Context, kind masterdata.Kind, q, group, state string) ([]*masterdata.Record, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	all, err := s.repo.List(ctx, kind)
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	group = strings.TrimSpace(group)
	state = strings.TrimSpace(state)
	out := make([]*masterdata.Record, 0, len(all))
	for _, r := range all {
		if q != "" && !strings.Contains(strings.ToLower(r.Code), q) &&
			!strings.Contains(strings.ToLower(r.Name), q) &&
			!strings.Contains(strings.ToLower(r.NameEN), q) {
			continue
		}
		if group != "" && r.GroupCode != group {
			continue
		}
		if state != "" && string(r.State) != state {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (s *service) List(ctx context.Context, kind masterdata.Kind, q, group, state string, page, size int) ([]*masterdata.Record, int, error) {
	all, err := s.Export(ctx, kind, q, group, state)
	if err != nil {
		return nil, 0, err
	}
	total := len(all)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	lo := (page - 1) * size
	if lo > total {
		lo = total
	}
	hi := lo + size
	if hi > total {
		hi = total
	}
	return all[lo:hi], total, nil
}

// ---------------------------------------------------------------------------
// Lifecycle (R8, R11)
// ---------------------------------------------------------------------------

func (s *service) Deactivate(ctx context.Context, kind masterdata.Kind, code, reason, actor string) (*masterdata.Record, error) {
	return s.deactivate(ctx, kind, code, reason, actor, false)
}

func (s *service) ForceDeactivate(ctx context.Context, kind masterdata.Kind, code, reason, actor string) (*masterdata.Record, error) {
	return s.deactivate(ctx, kind, code, reason, actor, true)
}

func (s *service) deactivate(ctx context.Context, kind masterdata.Kind, code, reason, actor string, force bool) (*masterdata.Record, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	rec, err := s.repo.GetByCode(ctx, kind, code)
	if err != nil {
		return nil, err
	}
	if rec.State == masterdata.StateInactive {
		return nil, masterdata.ErrInactive
	}
	children, err := s.children(ctx, kind, code)
	if err != nil {
		return nil, err
	}
	if !force {
		if rec.RefCount > 0 {
			return nil, masterdata.ErrBlockedRefs
		}
		if len(children) > 0 {
			return nil, masterdata.ErrBlockedRefs
		}
	}
	now := nowRFC3339()
	rec.State = masterdata.StateInactive
	rec.DeactivatedBy = actor
	rec.DeactivatedAt = now
	rec.DeactivateReason = strings.TrimSpace(reason)
	rec.UpdatedAt = now
	rec.UpdatedBy = actor
	if err := s.repo.Upsert(ctx, rec); err != nil {
		return nil, err
	}
	// Forced deactivation cascades to active descendants (R8).
	if force {
		for _, ch := range children {
			if ch.State != masterdata.StateActive {
				continue
			}
			ch.State = masterdata.StateInactive
			ch.DeactivatedBy = actor
			ch.DeactivatedAt = now
			ch.DeactivateReason = "ngừng theo mã " + code
			ch.UpdatedAt = now
			ch.UpdatedBy = actor
			if err := s.repo.Upsert(ctx, ch); err != nil {
				return nil, err
			}
		}
	}
	return rec, nil
}

func (s *service) Activate(ctx context.Context, kind masterdata.Kind, code, actor string) (*masterdata.Record, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	rec, err := s.repo.GetByCode(ctx, kind, code)
	if err != nil {
		return nil, err
	}
	if rec.State == masterdata.StateActive {
		return nil, masterdata.ErrInactive
	}
	if rec.GroupCode != "" {
		parent, err := s.repo.GetByCode(ctx, kind, rec.GroupCode)
		if err != nil {
			return nil, err
		}
		if parent.State != masterdata.StateActive {
			return nil, &masterdata.ValidationError{Kind: kind, Code: code,
				MessageVn: "Mã nhóm đang ngừng hoạt động",
				MessageEn: "cannot activate: group is inactive"}
		}
	}
	now := nowRFC3339()
	rec.State = masterdata.StateActive
	rec.DeactivatedBy = ""
	rec.DeactivatedAt = ""
	rec.DeactivateReason = ""
	rec.UpdatedAt = now
	rec.UpdatedBy = actor
	if err := s.repo.Upsert(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *service) Delete(ctx context.Context, kind masterdata.Kind, code, actor string) error {
	if !kind.IsKind() {
		return masterdata.ErrUnknownKind
	}
	rec, err := s.repo.GetByCode(ctx, kind, code)
	if err != nil {
		return err
	}
	if rec.RefCount > 0 {
		return masterdata.ErrBlockedRefs
	}
	children, err := s.children(ctx, kind, code)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return masterdata.ErrBlockedRefs
	}
	return s.repo.Delete(ctx, rec.ID)
}

// ---------------------------------------------------------------------------
// Merge (R10)
// ---------------------------------------------------------------------------

func (s *service) Merge(ctx context.Context, kind masterdata.Kind, keepCode string, dupeCodes []string, actor string, dryRun bool) (*masterdata.MergeResult, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	keep, err := s.repo.GetByCode(ctx, kind, keepCode)
	if err != nil {
		return nil, err
	}
	if keep.State != masterdata.StateActive {
		return nil, &masterdata.ValidationError{Kind: kind, Code: keepCode,
			MessageVn: "Mã giữ lại phải đang hoạt động", MessageEn: "keep target must be active"}
	}

	result := &masterdata.MergeResult{Keep: keepCode, Impacted: map[string]int64{}, DryRun: dryRun}
	seen := map[string]bool{keepCode: true}
	now := nowRFC3339()

	for _, dc := range dupeCodes {
		dc = strings.TrimSpace(dc)
		if dc == "" || seen[dc] {
			continue
		}
		seen[dc] = true

		dupe, err := s.repo.GetByCode(ctx, kind, dc)
		if errors.Is(err, masterdata.ErrNotFound) {
			return nil, &masterdata.ValidationError{Kind: kind, Code: dc,
				MessageVn: "Mã cần gộp không tồn tại: " + dc,
				MessageEn: "merge target not found: " + dc}
		}
		if err != nil {
			return nil, err
		}
		if dupe.ID == keep.ID {
			continue
		}

		impacted, err := s.repointReferences(ctx, kind, dc, keepCode, actor, now, dryRun)
		if err != nil {
			return nil, err
		}
		result.Impacted[dc] = int64(impacted)

		if dryRun {
			result.Merged = append(result.Merged, dc)
			continue
		}
		keep.RefCount += dupe.RefCount
		dupe.State = masterdata.StateInactive
		dupe.DeactivatedBy = actor
		dupe.DeactivatedAt = now
		dupe.DeactivateReason = "gộp vào " + keepCode
		dupe.UpdatedAt = now
		dupe.UpdatedBy = actor
		if err := s.repo.Upsert(ctx, dupe); err != nil {
			return nil, err
		}
		result.Merged = append(result.Merged, dc)
	}
	if !dryRun {
		keep.UpdatedAt = now
		keep.UpdatedBy = actor
		if err := s.repo.Upsert(ctx, keep); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// repointReferences moves intra-module references from the merged-away code to
// the keep code: same-kind group pointers plus Extra attributes of other kinds
// (e.g. item.unit_code, customer_code, ...). Cross-module consumers are stubs;
// the seam is SetReferenceCount.
func (s *service) repointReferences(ctx context.Context, kind masterdata.Kind, from, to, actor, now string, dryRun bool) (int, error) {
	all, err := s.repo.List(ctx, kind)
	if err != nil {
		return 0, err
	}
	impacted := 0
	for _, other := range all {
		if other.Code == from || other.Code == to {
			continue
		}
		changed := false
		if other.GroupCode == from {
			if !dryRun {
				other.GroupCode = to
			}
			changed = true
		}
		for k, v := range other.Extra {
			if refExtraKeys[k] && v == from {
				if !dryRun {
					other.Extra[k] = to
				}
				changed = true
			}
		}
		if changed {
			if !dryRun {
				other.UpdatedAt = now
				other.UpdatedBy = actor
				if err := s.repo.Upsert(ctx, other); err != nil {
					return 0, err
				}
			}
			impacted++
		}
	}

	// Cross-kind Extra references (e.g. an employee or item pointing at the
	// customer/supplier being merged away).
	for _, scanKind := range masterdata.Kinds {
		if scanKind == kind {
			continue
		}
		others, err := s.repo.List(ctx, scanKind)
		if err != nil {
			return 0, err
		}
		for _, other := range others {
			changed := false
			for k, v := range other.Extra {
				if refExtraKeys[k] && v == from {
					if !dryRun {
						other.Extra[k] = to
					}
					changed = true
				}
			}
			if changed {
				if !dryRun {
					other.UpdatedAt = now
					other.UpdatedBy = actor
					if err := s.repo.Upsert(ctx, other); err != nil {
						return 0, err
					}
				}
				impacted++
			}
		}
	}
	return impacted, nil
}

// ---------------------------------------------------------------------------
// Import / Export (CSV)
// ---------------------------------------------------------------------------

var importExtraCols = []string{
	"tax_code", "id_number", "passport_no", "budget_unit_code", "email", "phone",
	"address", "shipping_address", "note", "item_type", "unit_code", "vat_rate_code",
	"purchase_account", "sale_account", "inventory_account", "revenue_account",
	"cost_account", "department", "position", "location", "manager", "swift_code",
	"iso_code", "rate", "tax_kind", "account",
}

func (s *service) ImportRows(ctx context.Context, kind masterdata.Kind, rows [][]string, actor string, dryRun bool) (*masterdata.ImportResult, error) {
	if !kind.IsKind() {
		return nil, masterdata.ErrUnknownKind
	}
	res := &masterdata.ImportResult{DryRun: dryRun}
	if len(rows) < 2 {
		res.Total = len(rows) - 1
		return res, nil
	}

	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	col := func(row []string, names ...string) string {
		for _, n := range names {
			if i, ok := idx[strings.ToLower(n)]; ok && i < len(row) {
				if v := strings.TrimSpace(row[i]); v != "" {
					return v
				}
			}
		}
		return ""
	}

	res.Total = len(rows) - 1
	for i, row := range rows[1:] {
		rec := &masterdata.Record{
			Kind:      kind,
			Code:      col(row, "code", "ma", "account_code"),
			Name:      col(row, "name", "ten"),
			NameEN:    col(row, "name_en", "ten_en"),
			GroupCode: col(row, "group_code", "ma_nhom", "parent"),
			Extra:     map[string]string{},
		}
		for _, key := range importExtraCols {
			if v := col(row, key); v != "" {
				rec.Extra[key] = v
			}
		}

		created, err := s.importOne(ctx, rec, actor, dryRun)
		if err != nil {
			msg := err.Error()
			var ve *masterdata.ValidationError
			if errors.As(err, &ve) {
				msg = ve.MessageEn
			}
			res.Errors = append(res.Errors, masterdata.RowError{Row: i + 2, Message: msg})
			continue
		}
		if created {
			res.Created++
		} else {
			res.Updated++
		}
	}
	return res, nil
}

func (s *service) importOne(ctx context.Context, rec *masterdata.Record, actor string, dryRun bool) (bool, error) {
	existing, err := s.repo.GetByCode(ctx, rec.Kind, rec.Code)
	if err == nil {
		if dryRun {
			merged := existing.Clone()
			applyPatch(merged, rec)
			if err := s.validate(ctx, merged); err != nil {
				return false, err
			}
			return false, nil
		}
		if _, err := s.Update(ctx, rec.Kind, rec.Code, rec, actor); err != nil {
			return false, err
		}
		return false, nil
	}
	if !errors.Is(err, masterdata.ErrNotFound) {
		return false, err
	}

	if dryRun {
		v := rec.Clone()
		if err := s.validate(ctx, v); err != nil {
			return false, err
		}
		return true, nil
	}
	if _, err := s.Create(ctx, rec.Kind, rec, actor); err != nil {
		return false, err
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Regime (R13)
// ---------------------------------------------------------------------------

func (s *service) GetRegime(ctx context.Context) (string, error) {
	r, err := s.repo.GetRegime(ctx)
	if err != nil {
		return "", err
	}
	if r == "" {
		return DefaultRegime, nil
	}
	return r, nil
}

func (s *service) SetRegime(ctx context.Context, regime, actor string) error {
	regime = strings.ToUpper(strings.TrimSpace(regime))
	if !supportedRegimes[regime] {
		return &masterdata.ValidationError{Kind: masterdata.KindAccount,
			MessageVn: "Chưa hỗ trợ chế độ " + regime, MessageEn: "unsupported regime: " + regime}
	}
	if err := s.repo.SetRegime(ctx, regime, actor); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// References (consumer seam)
// ---------------------------------------------------------------------------

func (s *service) References(ctx context.Context, kind masterdata.Kind, code string) (int64, error) {
	rec, err := s.repo.GetByCode(ctx, kind, code)
	if err != nil {
		return 0, err
	}
	return rec.RefCount, nil
}

func (s *service) SetReferenceCount(ctx context.Context, kind masterdata.Kind, code string, n int64) error {
	rec, err := s.repo.GetByCode(ctx, kind, code)
	if err != nil {
		return err
	}
	rec.RefCount = n
	rec.UpdatedAt = nowRFC3339()
	return s.repo.Upsert(ctx, rec)
}

// ---------------------------------------------------------------------------
// Chart of accounts seed (TT 99/2025, Phụ lục 2 — representative subset)
// ---------------------------------------------------------------------------

type accountSeed struct {
	Code   string
	Name   string
	Parent string
	Type   masterdata.AccountType
}

// SeedAccounts idempotently imports the baseline chart of accounts. It skips
// codes that already exist and returns the number created. The full statutory
// chart is a follow-up; this subset covers the module's bootstrapping needs.
func (s *service) SeedAccounts(ctx context.Context, actor string) (int, error) {
	created := 0
	for _, a := range accountSeeds {
		if _, err := s.repo.GetByCode(ctx, masterdata.KindAccount, a.Code); err == nil {
			continue
		} else if !errors.Is(err, masterdata.ErrNotFound) {
			return created, err
		}
		rec := &masterdata.Record{
			Kind:        masterdata.KindAccount,
			Code:        a.Code,
			Name:        a.Name,
			GroupCode:   a.Parent,
			State:       masterdata.StateActive,
			AccountType: a.Type,
			Level:       levelOf(a.Code),
			AllowPost:   !hasChildren(a.Code),
			Extra:       map[string]string{"regime": DefaultRegime},
		}
		if a.Parent != "" {
			if p, err := s.repo.GetByCode(ctx, masterdata.KindAccount, a.Parent); err == nil {
				rec.Level = p.Level + 1
			}
		}
		rec.ID = masterdata.RecordID(rec.Kind, rec.Code)
		now := nowRFC3339()
		rec.CreatedAt, rec.UpdatedAt = now, now
		rec.CreatedBy, rec.UpdatedBy = actor, actor
		if err := s.repo.Upsert(ctx, rec); err != nil {
			return created, err
		}
		created++
	}
	if created > 0 {
		if err := s.repo.SetRegime(ctx, DefaultRegime, actor); err != nil {
			return created, err
		}
	}
	return created, nil
}

func levelOf(code string) int {
	switch len(code) {
	case 1, 2, 3:
		return 0
	case 4:
		return 1
	case 5, 6, 7:
		return 2
	default:
		return 2
	}
}

var accountChildren = map[string]bool{}

func hasChildren(code string) bool {
	return accountChildren[code]
}

var accountSeeds = []accountSeed{
	// Loại 1 — Tài sản
	{"111", "Tiền mặt", "", masterdata.AccountTypeAsset},
	{"1111", "Tiền Việt Nam", "111", masterdata.AccountTypeAsset},
	{"1112", "Ngoại tệ", "111", masterdata.AccountTypeAsset},
	{"1113", "Vàng, bạc, kim khí quý", "111", masterdata.AccountTypeAsset},
	{"112", "Tiền gửi ngân hàng", "", masterdata.AccountTypeAsset},
	{"1121", "Tiền Việt Nam", "112", masterdata.AccountTypeAsset},
	{"1122", "Ngoại tệ", "112", masterdata.AccountTypeAsset},
	{"113", "Tiền đang chuyển", "", masterdata.AccountTypeAsset},
	{"121", "Đầu tư tài chính ngắn hạn", "", masterdata.AccountTypeAsset},
	{"128", "Đầu tư nắm giữ đến ngày đáo hạn", "", masterdata.AccountTypeAsset},
	{"131", "Phải thu của khách hàng", "", masterdata.AccountTypeAsset},
	{"133", "Thuế GTGT được khấu trừ", "", masterdata.AccountTypeAsset},
	{"1331", "Thuế GTGT được khấu trừ của hàng hóa, dịch vụ", "133", masterdata.AccountTypeAsset},
	{"1332", "Thuế GTGT được khấu trừ của TSCĐ", "133", masterdata.AccountTypeAsset},
	{"138", "Phải thu khác", "", masterdata.AccountTypeAsset},
	{"1381", "Phải thu khác", "138", masterdata.AccountTypeAsset},
	{"1388", "Dự phòng phải thu khó đòi", "138", masterdata.AccountTypeAsset},
	{"141", "Tạm ứng", "", masterdata.AccountTypeAsset},
	{"142", "Chi phí trả trước", "", masterdata.AccountTypeAsset},
	{"151", "Hàng mua đang đi đường", "", masterdata.AccountTypeAsset},
	{"152", "Nguyên liệu, vật liệu", "", masterdata.AccountTypeAsset},
	{"1521", "Nguyên liệu, vật liệu chính", "152", masterdata.AccountTypeAsset},
	{"1522", "Vật liệu phụ", "152", masterdata.AccountTypeAsset},
	{"1523", "Nhiên liệu", "152", masterdata.AccountTypeAsset},
	{"1524", "Phụ tùng thay thế", "152", masterdata.AccountTypeAsset},
	{"153", "Công cụ, dụng cụ", "", masterdata.AccountTypeAsset},
	{"1531", "Công cụ, dụng cụ", "153", masterdata.AccountTypeAsset},
	{"1532", "Bao bì luân chuyển", "153", masterdata.AccountTypeAsset},
	{"154", "Chi phí sản xuất, kinh doanh dở dang", "", masterdata.AccountTypeAsset},
	{"155", "Thành phẩm", "", masterdata.AccountTypeAsset},
	{"156", "Hàng hóa", "", masterdata.AccountTypeAsset},
	{"1561", "Hàng hóa", "156", masterdata.AccountTypeAsset},
	{"1565", "Hàng hóa bất động sản", "156", masterdata.AccountTypeAsset},
	{"157", "Hàng gửi đi bán", "", masterdata.AccountTypeAsset},
	{"158", "Hàng hóa kho bảo thuế", "", masterdata.AccountTypeAsset},
	{"211", "TSCĐ hữu hình", "", masterdata.AccountTypeAsset},
	{"2111", "Nhà cửa, vật kiến trúc", "211", masterdata.AccountTypeAsset},
	{"2112", "Máy móc, thiết bị", "211", masterdata.AccountTypeAsset},
	{"2113", "Phương tiện vận tải, truyền dẫn", "211", masterdata.AccountTypeAsset},
	{"2114", "Thiết bị, dụng cụ quản lý", "211", masterdata.AccountTypeAsset},
	{"2118", "TSCĐ khác", "211", masterdata.AccountTypeAsset},
	{"212", "TSCĐ thuê tài chính", "", masterdata.AccountTypeAsset},
	{"213", "TSCĐ vô hình", "", masterdata.AccountTypeAsset},
	{"2131", "TSCĐ vô hình", "213", masterdata.AccountTypeAsset},
	{"214", "Hao mòn tài sản cố định", "", masterdata.AccountTypeAsset},
	{"2141", "Hao mòn TSCĐ hữu hình", "214", masterdata.AccountTypeAsset},
	{"2142", "Hao mòn TSCĐ thuê tài chính", "214", masterdata.AccountTypeAsset},
	{"2143", "Hao mòn TSCĐ vô hình", "214", masterdata.AccountTypeAsset},
	{"2147", "Hao mòn bất động sản đầu tư", "214", masterdata.AccountTypeAsset},
	{"217", "Bất động sản đầu tư", "", masterdata.AccountTypeAsset},
	{"221", "Đầu tư vào công ty con", "", masterdata.AccountTypeAsset},
	{"222", "Đầu tư vào công ty liên doanh, liên kết", "", masterdata.AccountTypeAsset},
	{"228", "Đầu tư khác", "", masterdata.AccountTypeAsset},
	{"229", "Dự phòng tổn thất tài sản", "", masterdata.AccountTypeAsset},
	{"2291", "Dự phòng giảm giá đầu tư ngắn hạn", "229", masterdata.AccountTypeAsset},
	{"2292", "Dự phòng tổn thất đầu tư dài hạn", "229", masterdata.AccountTypeAsset},
	{"2293", "Dự phòng phải thu khó đòi", "229", masterdata.AccountTypeAsset},
	{"2294", "Dự phòng giảm giá hàng tồn kho", "229", masterdata.AccountTypeAsset},
	{"241", "Xây dựng cơ bản dở dang", "", masterdata.AccountTypeAsset},
	{"2411", "Xây dựng cơ bản dở dang", "241", masterdata.AccountTypeAsset},
	{"2412", "Sửa chữa lớn TSCĐ", "241", masterdata.AccountTypeAsset},
	{"242", "Chi phí trả trước dài hạn", "", masterdata.AccountTypeAsset},
	{"243", "Tài sản thuế thu nhập hoãn lại", "", masterdata.AccountTypeAsset},
	{"244", "Ký quỹ, ký cược dài hạn", "", masterdata.AccountTypeAsset},

	// Loại 3 — Nợ phải trả
	{"311", "Vay và nợ thuê tài chính ngắn hạn", "", masterdata.AccountTypeLiability},
	{"315", "Nợ phải trả thuê tài chính", "", masterdata.AccountTypeLiability},
	{"331", "Phải trả cho người bán", "", masterdata.AccountTypeLiability},
	{"333", "Thuế và các khoản phải nộp Nhà nước", "", masterdata.AccountTypeLiability},
	{"3331", "Thuế GTGT phải nộp", "333", masterdata.AccountTypeLiability},
	{"3332", "Thuế thu nhập doanh nghiệp", "333", masterdata.AccountTypeLiability},
	{"3333", "Thuế tiêu thụ đặc biệt", "333", masterdata.AccountTypeLiability},
	{"3334", "Thuế thu nhập cá nhân", "333", masterdata.AccountTypeLiability},
	{"3335", "Thuế tài nguyên", "333", masterdata.AccountTypeLiability},
	{"3336", "Thuế sử dụng đất phi nông nghiệp", "333", masterdata.AccountTypeLiability},
	{"3337", "Thuế bảo vệ môi trường", "333", masterdata.AccountTypeLiability},
	{"3338", "Thuế nhà đất, tiền thuê đất", "333", masterdata.AccountTypeLiability},
	{"3339", "Phí, lệ phí và các khoản phải nộp khác", "333", masterdata.AccountTypeLiability},
	{"334", "Phải trả người lao động", "", masterdata.AccountTypeLiability},
	{"3341", "Phải trả người lao động", "334", masterdata.AccountTypeLiability},
	{"3348", "Phải trả người lao động khác", "334", masterdata.AccountTypeLiability},
	{"336", "Phải trả nội bộ", "", masterdata.AccountTypeLiability},
	{"337", "Thanh toán theo tiến độ kế hoạch hợp đồng xây dựng", "", masterdata.AccountTypeLiability},
	{"338", "Phải trả, phải nộp khác", "", masterdata.AccountTypeLiability},
	{"3381", "Tài sản thiếu chờ xử lý", "338", masterdata.AccountTypeLiability},
	{"3382", "Kinh phí công đoàn", "338", masterdata.AccountTypeLiability},
	{"3383", "Bảo hiểm xã hội", "338", masterdata.AccountTypeLiability},
	{"3384", "Bảo hiểm y tế", "338", masterdata.AccountTypeLiability},
	{"3385", "Phải trả về cổ phần hóa", "338", masterdata.AccountTypeLiability},
	{"3386", "Nhận ký quỹ, ký cược ngắn hạn", "338", masterdata.AccountTypeLiability},
	{"3388", "Phải trả, phải nộp khác", "338", masterdata.AccountTypeLiability},
	{"341", "Vay và nợ thuê tài chính dài hạn", "", masterdata.AccountTypeLiability},
	{"3411", "Vay dài hạn", "341", masterdata.AccountTypeLiability},
	{"3412", "Nợ thuê tài chính dài hạn", "341", masterdata.AccountTypeLiability},
	{"343", "Trái phiếu phát hành", "", masterdata.AccountTypeLiability},
	{"347", "Thuế thu nhập hoãn lại phải trả", "", masterdata.AccountTypeLiability},
	{"352", "Dự phòng phải trả", "", masterdata.AccountTypeLiability},

	// Loại 4 — Vốn chủ sở hữu
	{"411", "Vốn đầu tư của chủ sở hữu", "", masterdata.AccountTypeEquity},
	{"4111", "Vốn góp của chủ sở hữu", "411", masterdata.AccountTypeEquity},
	{"4112", "Thặng dư vốn cổ phần", "411", masterdata.AccountTypeEquity},
	{"4118", "Vốn khác", "411", masterdata.AccountTypeEquity},
	{"413", "Lợi thế thương mại", "", masterdata.AccountTypeEquity},
	{"419", "Cổ phiếu quỹ", "", masterdata.AccountTypeEquity},
	{"421", "Lợi nhuận sau thuế chưa phân phối", "", masterdata.AccountTypeEquity},
	{"4211", "Lợi nhuận sau thuế chưa phân phối năm trước", "421", masterdata.AccountTypeEquity},
	{"4212", "Lợi nhuận sau thuế chưa phân phối năm nay", "421", masterdata.AccountTypeEquity},
	{"431", "Quỹ khen thưởng, phúc lợi", "", masterdata.AccountTypeEquity},
	{"4311", "Quỹ khen thưởng", "431", masterdata.AccountTypeEquity},
	{"4312", "Quỹ phúc lợi", "431", masterdata.AccountTypeEquity},
	{"441", "Nguồn vốn đầu tư xây dựng cơ bản", "", masterdata.AccountTypeEquity},
	{"461", "Nguồn vốn khác", "", masterdata.AccountTypeEquity},

	// Loại 5 — Doanh thu
	{"511", "Doanh thu bán hàng và cung cấp dịch vụ", "", masterdata.AccountTypeRevenue},
	{"5111", "Doanh thu bán hàng hóa", "511", masterdata.AccountTypeRevenue},
	{"5112", "Doanh thu bán thành phẩm", "511", masterdata.AccountTypeRevenue},
	{"5113", "Doanh thu cung cấp dịch vụ", "511", masterdata.AccountTypeRevenue},
	{"5118", "Doanh thu khác", "511", masterdata.AccountTypeRevenue},
	{"515", "Doanh thu hoạt động tài chính", "", masterdata.AccountTypeRevenue},
	{"521", "Các khoản giảm trừ doanh thu", "", masterdata.AccountTypeRevenue},
	{"5211", "Chiết khấu thương mại", "521", masterdata.AccountTypeRevenue},
	{"5212", "Hàng bán bị trả lại", "521", masterdata.AccountTypeRevenue},
	{"5213", "Giảm giá hàng bán", "521", masterdata.AccountTypeRevenue},
	{"711", "Thu nhập khác", "", masterdata.AccountTypeRevenue},

	// Loại 6 — Chi phí sản xuất, kinh doanh
	{"611", "Mua hàng", "", masterdata.AccountTypeExpense},
	{"621", "Chi phí nguyên liệu, vật liệu trực tiếp", "", masterdata.AccountTypeExpense},
	{"6211", "Chi phí nguyên liệu, vật liệu trực tiếp", "621", masterdata.AccountTypeExpense},
	{"622", "Chi phí nhân công trực tiếp", "", masterdata.AccountTypeExpense},
	{"6221", "Chi phí nhân công trực tiếp", "622", masterdata.AccountTypeExpense},
	{"623", "Chi phí sử dụng máy thi công", "", masterdata.AccountTypeExpense},
	{"627", "Chi phí sản xuất chung", "", masterdata.AccountTypeExpense},
	{"6271", "Chi phí nhân viên phân xưởng", "627", masterdata.AccountTypeExpense},
	{"6272", "Chi phí vật liệu", "627", masterdata.AccountTypeExpense},
	{"6273", "Chi phí dụng cụ sản xuất", "627", masterdata.AccountTypeExpense},
	{"6274", "Chi phí khấu hao TSCĐ", "627", masterdata.AccountTypeExpense},
	{"6277", "Chi phí dịch vụ mua ngoài", "627", masterdata.AccountTypeExpense},
	{"6278", "Chi phí bằng tiền khác", "627", masterdata.AccountTypeExpense},
	{"631", "Giá thành sản xuất", "", masterdata.AccountTypeExpense},
	{"632", "Giá vốn hàng bán", "", masterdata.AccountTypeExpense},
	{"635", "Chi phí tài chính", "", masterdata.AccountTypeExpense},
	{"641", "Chi phí bán hàng", "", masterdata.AccountTypeExpense},
	{"6411", "Chi phí nhân viên bán hàng", "641", masterdata.AccountTypeExpense},
	{"6412", "Chi phí vật liệu, bao bì", "641", masterdata.AccountTypeExpense},
	{"6413", "Chi phí dụng cụ, đồ dùng", "641", masterdata.AccountTypeExpense},
	{"6414", "Chi phí khấu hao TSCĐ", "641", masterdata.AccountTypeExpense},
	{"6417", "Chi phí dịch vụ mua ngoài", "641", masterdata.AccountTypeExpense},
	{"6418", "Chi phí bằng tiền khác", "641", masterdata.AccountTypeExpense},
	{"642", "Chi phí quản lý doanh nghiệp", "", masterdata.AccountTypeExpense},
	{"6421", "Chi phí nhân viên quản lý", "642", masterdata.AccountTypeExpense},
	{"6422", "Chi phí vật liệu quản lý", "642", masterdata.AccountTypeExpense},
	{"6423", "Chi phí đồ dùng văn phòng", "642", masterdata.AccountTypeExpense},
	{"6424", "Chi phí khấu hao TSCĐ", "642", masterdata.AccountTypeExpense},
	{"6425", "Thuế, phí, lệ phí", "642", masterdata.AccountTypeExpense},
	{"6426", "Chi phí dự phòng", "642", masterdata.AccountTypeExpense},
	{"6427", "Chi phí dịch vụ mua ngoài", "642", masterdata.AccountTypeExpense},
	{"6428", "Chi phí bằng tiền khác", "642", masterdata.AccountTypeExpense},

	// Loại 8 — Chi phí khác
	{"811", "Chi phí khác", "", masterdata.AccountTypeExpense},
	{"821", "Chi phí thuế thu nhập doanh nghiệp", "", masterdata.AccountTypeExpense},
	{"8211", "Chi phí thuế thu nhập doanh nghiệp hiện hành", "821", masterdata.AccountTypeExpense},
	{"8212", "Chi phí thuế thu nhập doanh nghiệp hoãn lại", "821", masterdata.AccountTypeExpense},

	// Loại 9 — Xác định kết quả kinh doanh
	{"911", "Xác định kết quả kinh doanh", "", masterdata.AccountTypeResult},
}

func init() {
	children := map[string]bool{}
	for _, a := range accountSeeds {
		if a.Parent != "" {
			children[a.Parent] = true
		}
	}
	accountChildren = children
}
