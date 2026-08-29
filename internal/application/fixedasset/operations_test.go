package fixedasset

import (
	"testing"

	"goGL/internal/domain/fixedasset"
)

func TestTransfer_Success(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State:    fixedasset.StateActive,
		Location: "Building A",
	}
	err := Transfer(a, "Building B", "IT")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if a.Location != "Building B" {
		t.Errorf("location = %q, want Building B", a.Location)
	}
	if a.Department != "IT" {
		t.Errorf("department = %q, want IT", a.Department)
	}
}

func TestTransfer_InactiveRejects(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State:    fixedasset.StateInactive,
		Location: "Building A",
	}
	err := Transfer(a, "Building B", "IT")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestTransfer_SoldRejects(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State:    fixedasset.StateSold,
		Location: "Building A",
	}
	err := Transfer(a, "Building B", "IT")
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestLiquidate_Success(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateActive,
	}
	err := Liquidate(a)
	if err != nil {
		t.Fatalf("Liquidate: %v", err)
	}
	if a.State != fixedasset.StatePendingLiquidation {
		t.Errorf("state = %q, want pending_liquidation", a.State)
	}
}

func TestLiquidate_AlreadyScrapped(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateScrapped,
	}
	err := Liquidate(a)
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestLiquidate_AlreadySold(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateSold,
	}
	err := Liquidate(a)
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestConfirmLiquidation_Success_Scrapped(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StatePendingLiquidation,
	}
	err := ConfirmLiquidation(a, fixedasset.StateScrapped)
	if err != nil {
		t.Fatalf("ConfirmLiquidation: %v", err)
	}
	if a.State != fixedasset.StateScrapped {
		t.Errorf("state = %q, want scrapped", a.State)
	}
}

func TestConfirmLiquidation_Success_Sold(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StatePendingLiquidation,
	}
	err := ConfirmLiquidation(a, fixedasset.StateSold)
	if err != nil {
		t.Fatalf("ConfirmLiquidation: %v", err)
	}
	if a.State != fixedasset.StateSold {
		t.Errorf("state = %q, want sold", a.State)
	}
}

func TestConfirmLiquidation_NotPending(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateActive,
	}
	err := ConfirmLiquidation(a, fixedasset.StateScrapped)
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestConfirmLiquidation_InvalidState(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StatePendingLiquidation,
	}
	err := ConfirmLiquidation(a, fixedasset.StateActive)
	if err != fixedasset.ErrInvalid {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDeactivate_Success(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateActive,
	}
	err := Deactivate(a)
	if err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	if a.State != fixedasset.StateInactive {
		t.Errorf("state = %q, want inactive", a.State)
	}
}

func TestDeactivate_NotActive(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateInactive,
	}
	err := Deactivate(a)
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestReactivate_Success(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateInactive,
	}
	err := Reactivate(a)
	if err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if a.State != fixedasset.StateActive {
		t.Errorf("state = %q, want active", a.State)
	}
}

func TestReactivate_NotInactive(t *testing.T) {
	a := &fixedasset.FixedAsset{
		State: fixedasset.StateActive,
	}
	err := Reactivate(a)
	if err != fixedasset.ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}
