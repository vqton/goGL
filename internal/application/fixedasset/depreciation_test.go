package fixedasset

import (
	"testing"

	"goGL/internal/domain/fixedasset"
)

func TestDepreciationEngine_StraightLine(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000, // 120M VND
		ResidualValue:      12_000_000,  // 12M VND
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120, // 10 years
	}
	// (120M - 12M) / 120 = 900,000 VND per month
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	if monthly != 900_000 {
		t.Errorf("monthly = %d, want 900000", monthly)
	}
}

func TestDepreciationEngine_DecliningBalance(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		AccumulatedDepr:    0,
		DepreciationMethod: fixedasset.MethodDeclining,
		UsefulLifeMonths:   120,
	}
	// 120M / 120 = 1,000,000 per month (first month)
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	if monthly != 1_000_000 {
		t.Errorf("monthly = %d, want 1000000", monthly)
	}
}

func TestDepreciationEngine_DecliningBalance_WithAccumulated(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		AccumulatedDepr:    30_000_000,
		DepreciationMethod: fixedasset.MethodDeclining,
		UsefulLifeMonths:   120,
	}
	// (120M - 30M) / 120 = 750,000 per month
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	if monthly != 750_000 {
		t.Errorf("monthly = %d, want 750000", monthly)
	}
}

func TestDepreciationEngine_UnitsOfOutput(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		DepreciationMethod: fixedasset.MethodUnitsOfOutput,
		UsefulLifeMonths:   120,
	}
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	if monthly != 900_000 {
		t.Errorf("monthly = %d, want 900000", monthly)
	}
}

func TestDepreciationEngine_FullyDepreciated(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		AccumulatedDepr:    108_000_000, // Fully depreciated
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120,
	}
	_, err := engine.MonthlyDepreciation(a)
	if err != fixedasset.ErrFullyDepreciated {
		t.Errorf("expected ErrFullyDepreciated, got %v", err)
	}
}

func TestDepreciationEngine_CapsAtMaxDepreciation(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		AccumulatedDepr:    107_000_000, // Only 1M left
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120,
	}
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	// Normal would be 900k, but cap at remaining 1M
	if monthly != 900_000 {
		t.Errorf("monthly = %d, want 900000", monthly)
	}
}

func TestDepreciationEngine_CapAtResidual(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		AccumulatedDepr:    107_500_000, // 500k left before residual
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120,
	}
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	if monthly != 500_000 {
		t.Errorf("monthly = %d, want 500000", monthly)
	}
}

func TestDepreciationEngine_CalculatePeriodDepreciation(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120,
	}
	total, updated, err := engine.CalculatePeriodDepreciation(a, 12)
	if err != nil {
		t.Fatalf("CalculatePeriodDepreciation: %v", err)
	}
	// 900k * 12 = 10,800,000
	if total != 10_800_000 {
		t.Errorf("total = %d, want 10800000", total)
	}
	if updated.AccumulatedDepr != 10_800_000 {
		t.Errorf("accum = %d, want 10800000", updated.AccumulatedDepr)
	}
	// Verify original asset is NOT mutated
	if a.AccumulatedDepr != 0 {
		t.Errorf("input mutated: accum = %d, want 0", a.AccumulatedDepr)
	}
}

func TestDepreciationEngine_CalculatePeriodDepreciation_NoMutateInput(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   120,
	}
	originalAccum := a.AccumulatedDepr
	_, _, err := engine.CalculatePeriodDepreciation(a, 6)
	if err != nil {
		t.Fatalf("CalculatePeriodDepreciation: %v", err)
	}
	if a.AccumulatedDepr != originalAccum {
		t.Errorf("input mutated: accum = %d, want %d", a.AccumulatedDepr, originalAccum)
	}
}

func TestDepreciationEngine_InvalidMethod(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		DepreciationMethod: "invalid",
		UsefulLifeMonths:   120,
	}
	_, err := engine.MonthlyDepreciation(a)
	if err != fixedasset.ErrInvalid {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestDepreciationEngine_ZeroUsefulLife(t *testing.T) {
	engine := NewDepreciationEngine()
	a := &fixedasset.FixedAsset{
		OriginalCost:       120_000_000,
		ResidualValue:      12_000_000,
		DepreciationMethod: fixedasset.MethodStraightLine,
		UsefulLifeMonths:   0,
	}
	monthly, err := engine.MonthlyDepreciation(a)
	if err != nil {
		t.Fatalf("MonthlyDepreciation: %v", err)
	}
	if monthly != 0 {
		t.Errorf("monthly = %d, want 0", monthly)
	}
}
