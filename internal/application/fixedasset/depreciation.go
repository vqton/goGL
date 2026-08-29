package fixedasset

import (
	"goGL/internal/domain/fixedasset"
)

// DepreciationEngine calculates monthly depreciation for fixed assets.
type DepreciationEngine struct{}

func NewDepreciationEngine() *DepreciationEngine {
	return &DepreciationEngine{}
}

// MonthlyDepreciation calculates one month's depreciation amount for the given asset.
func (e *DepreciationEngine) MonthlyDepreciation(a *fixedasset.FixedAsset) (int64, error) {
	if !a.DepreciationMethod.IsValid() {
		return 0, fixedasset.ErrInvalid
	}
	if a.IsFullyDepreciated() {
		return 0, fixedasset.ErrFullyDepreciated
	}

	depreciableValue := a.OriginalCost - a.ResidualValue
	maxDepreciation := depreciableValue - a.AccumulatedDepr

	var amount int64
	switch a.DepreciationMethod {
	case fixedasset.MethodStraightLine:
		amount = e.straightLine(depreciableValue, a.UsefulLifeMonths)
	case fixedasset.MethodDeclining:
		amount = e.decliningBalance(a.OriginalCost, a.AccumulatedDepr, a.UsefulLifeMonths)
	case fixedasset.MethodUnitsOfOutput:
		amount = e.unitsOfOutput(depreciableValue, a.UsefulLifeMonths)
	}

	if amount > maxDepreciation {
		amount = maxDepreciation
	}
	if amount < 0 {
		amount = 0
	}
	return amount, nil
}

// straightLine: (Cost - Residual) / Useful Life (months)
func (e *DepreciationEngine) straightLine(depreciableValue int64, usefulLifeMonths int) int64 {
	if usefulLifeMonths <= 0 {
		return 0
	}
	return depreciableValue / int64(usefulLifeMonths)
}

// decliningBalance: (Original Cost - Accumulated) / Remaining Life
// Vietnamese approach: uniform allocation over useful life
func (e *DepreciationEngine) decliningBalance(originalCost, accumulatedDepr int64, usefulLifeMonths int) int64 {
	if usefulLifeMonths <= 0 {
		return 0
	}
	bookValue := originalCost - accumulatedDepr
	return bookValue / int64(usefulLifeMonths)
}

// unitsOfOutput: (Cost - Residual) / Total Estimated Units * Units This Period
// For monthly calculation, we use uniform distribution over useful life.
func (e *DepreciationEngine) unitsOfOutput(depreciableValue int64, usefulLifeMonths int) int64 {
	if usefulLifeMonths <= 0 {
		return 0
	}
	return depreciableValue / int64(usefulLifeMonths)
}

// CalculatePeriodDepreciation calculates depreciation for a period (multiple months).
func (e *DepreciationEngine) CalculatePeriodDepreciation(a *fixedasset.FixedAsset, months int) (int64, error) {
	total := int64(0)
	for i := 0; i < months; i++ {
		monthly, err := e.MonthlyDepreciation(a)
		if err != nil {
			return total, err
		}
		total += monthly
		a.AccumulatedDepr += monthly
	}
	return total, nil
}
