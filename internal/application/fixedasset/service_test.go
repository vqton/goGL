package fixedasset

import (
	"context"
	"testing"

	"goGL/internal/domain/fixedasset"
)

type stubRepo struct {
	assets map[string]*fixedasset.FixedAsset
	seq    int64
}

func newStubRepo() *stubRepo {
	return &stubRepo{assets: make(map[string]*fixedasset.FixedAsset)}
}

func (r *stubRepo) Create(_ context.Context, a *fixedasset.FixedAsset) error {
	cp := *a
	r.assets[a.ID] = &cp
	return nil
}

func (r *stubRepo) FindByID(_ context.Context, id string) (*fixedasset.FixedAsset, error) {
	a, ok := r.assets[id]
	if !ok {
		return nil, fixedasset.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *stubRepo) Update(_ context.Context, a *fixedasset.FixedAsset) error {
	r.assets[a.ID] = a
	return nil
}

func (r *stubRepo) Delete(_ context.Context, id string) error {
	delete(r.assets, id)
	return nil
}

func (r *stubRepo) List(_ context.Context, _ fixedasset.AssetType, _ fixedasset.AssetState, _, _ int) ([]*fixedasset.FixedAsset, error) {
	var out []*fixedasset.FixedAsset
	for _, a := range r.assets {
		out = append(out, a)
	}
	return out, nil
}

func (r *stubRepo) NextCode(_ context.Context) (int64, error) {
	r.seq++
	return r.seq, nil
}

func testAsset() *fixedasset.FixedAsset {
	return &fixedasset.FixedAsset{
		ID:                 "asset-1",
		Name:               "Test Machine",
		AssetType:          fixedasset.TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
		State:              fixedasset.StateActive,
	}
}

func TestService_Create_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()

	if err := svc.Create(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.Code != "FA-00001" {
		t.Errorf("code = %q, want FA-00001", a.Code)
	}
	if a.State != fixedasset.StateActive {
		t.Errorf("state = %q, want active", a.State)
	}
}

func TestService_Create_ValidationFails(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := &fixedasset.FixedAsset{} // missing name, cost, etc.

	err := svc.Create(context.Background(), a)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestService_Create_IncrementsCode(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)

	a1 := testAsset()
	a1.ID = "asset-1"
	if err := svc.Create(context.Background(), a1); err != nil {
		t.Fatalf("Create a1: %v", err)
	}

	a2 := testAsset()
	a2.ID = "asset-2"
	if err := svc.Create(context.Background(), a2); err != nil {
		t.Fatalf("Create a2: %v", err)
	}

	if a2.Code != "FA-00002" {
		t.Errorf("code = %q, want FA-00002", a2.Code)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	got, err := svc.GetByID(context.Background(), "asset-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Test Machine" {
		t.Errorf("name = %q, want Test Machine", got.Name)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
}

func TestService_Update_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	update := a.Clone()
	update.Name = "Updated Machine"
	if err := svc.Update(context.Background(), update); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := svc.GetByID(context.Background(), "asset-1")
	if got.Name != "Updated Machine" {
		t.Errorf("name = %q, want Updated Machine", got.Name)
	}
}

func TestService_Delete_Inactive(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StateInactive
	repo.Create(context.Background(), a)

	if err := svc.Delete(context.Background(), "asset-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestService_Delete_ActiveRejects(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	// Create sets state to Active
	if err := svc.Create(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Logf("state after create: %q", a.State)

	err := svc.Delete(context.Background(), "asset-1")
	t.Logf("delete error: %v", err)
	if err == nil {
		t.Fatal("expected error deleting active asset")
	}
}

func TestService_List_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	list, err := svc.List(context.Background(), "", "", 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1", len(list))
	}
}

func TestService_Transfer_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	got, err := svc.Transfer(context.Background(), "asset-1", "Building B", "IT")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if got.Location != "Building B" {
		t.Errorf("location = %q, want Building B", got.Location)
	}
	if got.Department != "IT" {
		t.Errorf("department = %q, want IT", got.Department)
	}
}

func TestService_Transfer_InactiveRejects(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StateInactive
	repo.Create(context.Background(), a)

	_, err := svc.Transfer(context.Background(), "asset-1", "Building B", "IT")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestService_Liquidate_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	got, err := svc.Liquidate(context.Background(), "asset-1")
	if err != nil {
		t.Fatalf("Liquidate: %v", err)
	}
	if got.State != fixedasset.StatePendingLiquidation {
		t.Errorf("state = %q, want pending_liquidation", got.State)
	}
}

func TestService_ConfirmLiquidation_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StatePendingLiquidation
	repo.Create(context.Background(), a)

	got, err := svc.ConfirmLiquidation(context.Background(), "asset-1", fixedasset.StateScrapped)
	if err != nil {
		t.Fatalf("ConfirmLiquidation: %v", err)
	}
	if got.State != fixedasset.StateScrapped {
		t.Errorf("state = %q, want scrapped", got.State)
	}
}

func TestService_Deactivate_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	got, err := svc.Deactivate(context.Background(), "asset-1")
	if err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	if got.State != fixedasset.StateInactive {
		t.Errorf("state = %q, want inactive", got.State)
	}
}

func TestService_Reactivate_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StateInactive
	repo.Create(context.Background(), a)

	got, err := svc.Reactivate(context.Background(), "asset-1")
	if err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if got.State != fixedasset.StateActive {
		t.Errorf("state = %q, want active", got.State)
	}
}

// --- Depreciation entry stub ---

type stubDeprRepo struct {
	entries map[string]*fixedasset.DepreciationEntry
	posted  map[string]bool
}

func newStubDeprRepo() *stubDeprRepo {
	return &stubDeprRepo{
		entries: make(map[string]*fixedasset.DepreciationEntry),
		posted:  make(map[string]bool),
	}
}

func (r *stubDeprRepo) Create(_ context.Context, e *fixedasset.DepreciationEntry) error {
	cp := *e
	r.entries[e.ID] = &cp
	return nil
}

func (r *stubDeprRepo) FindByID(_ context.Context, id string) (*fixedasset.DepreciationEntry, error) {
	e, ok := r.entries[id]
	if !ok {
		return nil, fixedasset.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (r *stubDeprRepo) Update(_ context.Context, e *fixedasset.DepreciationEntry) error {
	r.entries[e.ID] = e
	return nil
}

func (r *stubDeprRepo) ListByAsset(_ context.Context, assetID string) ([]*fixedasset.DepreciationEntry, error) {
	var out []*fixedasset.DepreciationEntry
	for _, e := range r.entries {
		if e.AssetID == assetID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *stubDeprRepo) ListByPeriod(_ context.Context, period string) ([]*fixedasset.DepreciationEntry, error) {
	var out []*fixedasset.DepreciationEntry
	for _, e := range r.entries {
		if e.Period == period {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *stubDeprRepo) FindByAssetAndPeriod(_ context.Context, assetID, period string) (*fixedasset.DepreciationEntry, error) {
	for _, e := range r.entries {
		if e.AssetID == assetID && e.Period == period {
			cp := *e
			return &cp, nil
		}
	}
	return nil, fixedasset.ErrNotFound
}

func (r *stubDeprRepo) IsPeriodPosted(_ context.Context, period string) (bool, error) {
	return r.posted[period], nil
}

// --- Depreciation service tests ---

func TestService_RunMonthlyDepreciation_Success(t *testing.T) {
	repo := newStubRepo()
	deprRepo := newStubDeprRepo()
	svc := NewServiceWithDepreciation(repo, deprRepo)

	a := testAsset()
	repo.Create(context.Background(), a)

	entries, err := svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")
	if err != nil {
		t.Fatalf("RunMonthlyDepreciation: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.AssetID != "asset-1" {
		t.Errorf("AssetID = %q, want asset-1", e.AssetID)
	}
	if e.Period != "2026-08" {
		t.Errorf("Period = %q, want 2026-08", e.Period)
	}
	if e.Amount != 375000 { // (50M - 5M) / 120 = 375000
		t.Errorf("Amount = %d, want 375000", e.Amount)
	}
	if e.AccountDebit != "627" {
		t.Errorf("AccountDebit = %q, want 627", e.AccountDebit)
	}
	if e.AccountCredit != "2141" {
		t.Errorf("AccountCredit = %q, want 2141", e.AccountCredit)
	}
	if e.Status != fixedasset.DepreciationDraft {
		t.Errorf("Status = %q, want draft", e.Status)
	}
}

func TestService_RunMonthlyDepreciation_AlreadyPosted(t *testing.T) {
	repo := newStubRepo()
	deprRepo := newStubDeprRepo()
	svc := NewServiceWithDepreciation(repo, deprRepo)

	a := testAsset()
	repo.Create(context.Background(), a)
	deprRepo.posted["2026-08"] = true

	_, err := svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestService_RunMonthlyDepreciation_NoDepreciationRepo(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo) // no deprRepo

	_, err := svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")
	if err != fixedasset.ErrInvalid {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestService_RunMonthlyDepreciation_SkipsFullyDepreciated(t *testing.T) {
	repo := newStubRepo()
	deprRepo := newStubDeprRepo()
	svc := NewServiceWithDepreciation(repo, deprRepo)

	a := testAsset()
	a.AccumulatedDepr = 45_000_000 // already almost fully depreciated
	repo.Create(context.Background(), a)

	entries, err := svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")
	if err != nil {
		t.Fatalf("RunMonthlyDepreciation: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for fully depreciated asset, got %d", len(entries))
	}
}

func TestService_GetDepreciationSchedule_Success(t *testing.T) {
	repo := newStubRepo()
	deprRepo := newStubDeprRepo()
	svc := NewServiceWithDepreciation(repo, deprRepo)

	a := testAsset()
	repo.Create(context.Background(), a)
	svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")

	schedule, err := svc.GetDepreciationSchedule(context.Background(), "asset-1")
	if err != nil {
		t.Fatalf("GetDepreciationSchedule: %v", err)
	}
	if len(schedule) != 1 {
		t.Errorf("expected 1 entry, got %d", len(schedule))
	}
}

func TestService_GetDepreciationByPeriod_Success(t *testing.T) {
	repo := newStubRepo()
	deprRepo := newStubDeprRepo()
	svc := NewServiceWithDepreciation(repo, deprRepo)

	a := testAsset()
	repo.Create(context.Background(), a)
	svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")

	entries, err := svc.GetDepreciationByPeriod(context.Background(), "2026-08")
	if err != nil {
		t.Fatalf("GetDepreciationByPeriod: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestService_RunMonthlyDepreciation_MultipleAssets(t *testing.T) {
	repo := newStubRepo()
	deprRepo := newStubDeprRepo()
	svc := NewServiceWithDepreciation(repo, deprRepo)

	a1 := testAsset()
	a1.ID = "asset-1"
	repo.Create(context.Background(), a1)

	a2 := testAsset()
	a2.ID = "asset-2"
	a2.OriginalCost = 100_000_000
	a2.UsefulLifeMonths = 240
	repo.Create(context.Background(), a2)

	entries, err := svc.RunMonthlyDepreciation(context.Background(), "2026-08", "admin")
	if err != nil {
		t.Fatalf("RunMonthlyDepreciation: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

// --- Approval workflow tests ---

func TestService_Liquidate_SetsApprovalPending(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	repo.Create(context.Background(), a)

	got, err := svc.Liquidate(context.Background(), "asset-1")
	if err != nil {
		t.Fatalf("Liquidate: %v", err)
	}
	if got.ApprovalStatus != fixedasset.ApprovalPending {
		t.Errorf("ApprovalStatus = %q, want pending", got.ApprovalStatus)
	}
}

func TestService_ApproveLiquidation_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StatePendingLiquidation
	a.ApprovalStatus = fixedasset.ApprovalPending
	repo.Create(context.Background(), a)

	got, err := svc.ApproveLiquidation(context.Background(), "asset-1", "director")
	if err != nil {
		t.Fatalf("ApproveLiquidation: %v", err)
	}
	if got.ApprovalStatus != fixedasset.ApprovalApproved {
		t.Errorf("ApprovalStatus = %q, want approved", got.ApprovalStatus)
	}
	if got.ApprovedBy != "director" {
		t.Errorf("ApprovedBy = %q, want director", got.ApprovedBy)
	}
	if got.ApprovedAt == "" {
		t.Error("ApprovedAt should be set")
	}
}

func TestService_ApproveLiquidation_WrongState(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StateActive
	repo.Create(context.Background(), a)

	_, err := svc.ApproveLiquidation(context.Background(), "asset-1", "director")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestService_ApproveLiquidation_NotPending(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StatePendingLiquidation
	a.ApprovalStatus = fixedasset.ApprovalNone // no approval request
	repo.Create(context.Background(), a)

	_, err := svc.ApproveLiquidation(context.Background(), "asset-1", "director")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestService_RejectLiquidation_Success(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StatePendingLiquidation
	a.ApprovalStatus = fixedasset.ApprovalPending
	repo.Create(context.Background(), a)

	got, err := svc.RejectLiquidation(context.Background(), "asset-1", "director", "Asset still in use")
	if err != nil {
		t.Fatalf("RejectLiquidation: %v", err)
	}
	if got.ApprovalStatus != fixedasset.ApprovalRejected {
		t.Errorf("ApprovalStatus = %q, want rejected", got.ApprovalStatus)
	}
	if got.RejectReason != "Asset still in use" {
		t.Errorf("RejectReason = %q, want 'Asset still in use'", got.RejectReason)
	}
	if got.State != fixedasset.StateActive {
		t.Errorf("State = %q, want active (reverted)", got.State)
	}
}

func TestService_RejectLiquidation_WrongState(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo)
	a := testAsset()
	a.State = fixedasset.StateActive
	repo.Create(context.Background(), a)

	_, err := svc.RejectLiquidation(context.Background(), "asset-1", "director", "reason")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}
