package fixedasset

import (
	"errors"
	"testing"

	"goGL/internal/domain/core"
)

func TestValidateAsset_Success(t *testing.T) {
	a := &FixedAsset{
		Name:               "Machine A",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	if err := ValidateAsset(a); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if a.State != StateActive {
		t.Errorf("state = %q, want active", a.State)
	}
}

func TestValidateAsset_EmptyName(t *testing.T) {
	a := &FixedAsset{
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for empty name")
	}
	var ve *core.ValidationError
	if !errors.As(err, &ve) || ve.Field != "name" {
		t.Errorf("expected ValidationError for name, got %v", err)
	}
}

func TestValidateAsset_CostTooLow(t *testing.T) {
	a := &FixedAsset{
		Name:               "Cheap Item",
		AssetType:          TypeTools,
		OriginalCost:       10_000_000, // Below 30M threshold
		ResidualValue:      1_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   36,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for cost below 30M")
	}
	var ve *core.ValidationError
	if !errors.As(err, &ve) || ve.Field != "original_cost" {
		t.Errorf("expected ValidationError for original_cost, got %v", err)
	}
}

func TestValidateAsset_ResidualExceedsCost(t *testing.T) {
	a := &FixedAsset{
		Name:               "Bad Residual",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      50_000_000, // Equal to cost
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for residual >= cost")
	}
}

func TestValidateAsset_NegativeResidual(t *testing.T) {
	a := &FixedAsset{
		Name:               "Negative Residual",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      -1_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for negative residual")
	}
}

func TestValidateAsset_ZeroUsefulLife(t *testing.T) {
	a := &FixedAsset{
		Name:               "Zero Life",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   0,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for zero useful life")
	}
}

func TestValidateAsset_InvalidDepreciationMethod(t *testing.T) {
	a := &FixedAsset{
		Name:               "Bad Method",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: "invalid",
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for invalid depreciation method")
	}
}

func TestValidateAsset_InvalidAssetType(t *testing.T) {
	a := &FixedAsset{
		Name:               "Bad Type",
		AssetType:          "invalid",
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for invalid asset type")
	}
}

func TestValidateAsset_MissingPurchaseDate(t *testing.T) {
	a := &FixedAsset{
		Name:               "No Date",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		InServiceDate:      "2026-01-15",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for missing purchase date")
	}
}

func TestValidateAsset_MissingInServiceDate(t *testing.T) {
	a := &FixedAsset{
		Name:               "No In-Service Date",
		AssetType:          TypeMachinery,
		OriginalCost:       50_000_000,
		ResidualValue:      5_000_000,
		DepreciationMethod: MethodStraightLine,
		UsefulLifeMonths:   120,
		PurchaseDate:       "2026-01-01",
	}
	err := ValidateAsset(a)
	if err == nil {
		t.Error("expected error for missing in-service date")
	}
}

func TestAssetState_IsValid(t *testing.T) {
	tests := []struct {
		state AssetState
		want  bool
	}{
		{StateActive, true},
		{StateInactive, true},
		{StateScrapped, true},
		{StateSold, true},
		{StatePendingLiquidation, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.state.IsValid(); got != tt.want {
			t.Errorf("AssetState(%q).IsValid() = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestDepreciationMethod_IsValid(t *testing.T) {
	tests := []struct {
		method DepreciationMethod
		want   bool
	}{
		{MethodStraightLine, true},
		{MethodDeclining, true},
		{MethodUnitsOfOutput, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.method.IsValid(); got != tt.want {
			t.Errorf("DepreciationMethod(%q).IsValid() = %v, want %v", tt.method, got, tt.want)
		}
	}
}

func TestAssetType_IsValid(t *testing.T) {
	tests := []struct {
		assetType AssetType
		want      bool
	}{
		{TypeHousing, true},
		{TypeMachinery, true},
		{TypeTransport, true},
		{TypeTools, true},
		{TypePerennial, true},
		{TypeOther, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.assetType.IsValid(); got != tt.want {
			t.Errorf("AssetType(%q).IsValid() = %v, want %v", tt.assetType, got, tt.want)
		}
	}
}

func TestDepreciationLifeRange(t *testing.T) {
	tests := []struct {
		assetType AssetType
		minMonths int
		maxMonths int
	}{
		{TypeHousing, 60, 600},
		{TypeMachinery, 36, 240},
		{TypeTransport, 72, 360},
		{TypeTools, 36, 120},
		{TypePerennial, 48, 480},
		{TypeOther, 48, 300},
	}
	for _, tt := range tests {
		minM, maxM := DepreciationLifeRange(tt.assetType)
		if minM != tt.minMonths || maxM != tt.maxMonths {
			t.Errorf("DepreciationLifeRange(%q) = (%d, %d), want (%d, %d)",
				tt.assetType, minM, maxM, tt.minMonths, tt.maxMonths)
		}
	}
}

func TestFixedAsset_CurrentValue(t *testing.T) {
	a := &FixedAsset{
		OriginalCost:    100_000_000,
		AccumulatedDepr: 40_000_000,
	}
	if got := a.CurrentValue(); got != 60_000_000 {
		t.Errorf("CurrentValue() = %d, want 60000000", got)
	}
}

func TestFixedAsset_IsFullyDepreciated(t *testing.T) {
	a := &FixedAsset{
		OriginalCost:    100_000_000,
		ResidualValue:   10_000_000,
		AccumulatedDepr: 90_000_000, // 100M - 10M = 90M depreciable
	}
	if !a.IsFullyDepreciated() {
		t.Error("expected fully depreciated")
	}

	a2 := &FixedAsset{
		OriginalCost:    100_000_000,
		ResidualValue:   10_000_000,
		AccumulatedDepr: 80_000_000, // Not yet
	}
	if a2.IsFullyDepreciated() {
		t.Error("expected not fully depreciated")
	}
}

func TestFixedAsset_Clone(t *testing.T) {
	a := &FixedAsset{
		ID:              "test-id",
		Code:            "FA-00001",
		Name:            "Test Asset",
		OriginalCost:    50_000_000,
		AccumulatedDepr: 20_000_000,
	}
	cp := a.Clone()
	cp.Name = "Modified"
	cp.OriginalCost = 100_000_000

	if a.Name != "Test Asset" {
		t.Error("clone modified original name")
	}
	if a.OriginalCost != 50_000_000 {
		t.Error("clone modified original cost")
	}
}
