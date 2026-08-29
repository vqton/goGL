package fixedasset

import (
	"testing"
)

func TestDepreciationStatusIsValid(t *testing.T) {
	tests := []struct {
		status DepreciationStatus
		valid  bool
	}{
		{DepreciationDraft, true},
		{DepreciationPosted, true},
		{DepreciationStatus("invalid"), false},
		{DepreciationStatus(""), false},
	}
	for _, tt := range tests {
		if got := tt.status.IsValid(); got != tt.valid {
			t.Errorf("DepreciationStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
		}
	}
}

func TestDepreciationEntry_Fields(t *testing.T) {
	e := DepreciationEntry{
		ID:                 "test-id",
		AssetID:            "asset-1",
		AssetCode:          "FA-00001",
		AssetName:          "Machine A",
		Period:             "2026-08",
		DepreciationMethod: MethodStraightLine,
		Amount:             4166667,
		AccumulatedDepr:    4166667,
		BookValue:          495833333,
		AccountDebit:       "627",
		AccountCredit:      "2141",
		Status:             DepreciationDraft,
		CreatedBy:          "admin",
		CreatedAt:          "2026-08-29T00:00:00Z",
	}

	if e.ID != "test-id" {
		t.Errorf("ID = %q, want test-id", e.ID)
	}
	if e.AssetID != "asset-1" {
		t.Errorf("AssetID = %q, want asset-1", e.AssetID)
	}
	if e.Period != "2026-08" {
		t.Errorf("Period = %q, want 2026-08", e.Period)
	}
	if e.Amount != 4166667 {
		t.Errorf("Amount = %d, want 4166667", e.Amount)
	}
	if e.Status != DepreciationDraft {
		t.Errorf("Status = %q, want %s", e.Status, DepreciationDraft)
	}
	if e.AccountDebit != "627" {
		t.Errorf("AccountDebit = %q, want 627", e.AccountDebit)
	}
	if e.AccountCredit != "2141" {
		t.Errorf("AccountCredit = %q, want 2141", e.AccountCredit)
	}
}
