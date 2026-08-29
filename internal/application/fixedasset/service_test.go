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
