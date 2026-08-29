package fixedasset

import (
	"goGL/internal/domain/fixedasset"
)

// Transfer transfers an asset to a new location/department.
func Transfer(a *fixedasset.FixedAsset, toLocation, toDepartment string) error {
	if a.State != fixedasset.StateActive {
		return fixedasset.ErrConflict
	}
	a.Location = toLocation
	a.Department = toDepartment
	a.UpdatedAt = fixedasset.NowRFC3339()
	return nil
}

// Liquidation initiates the liquidation process for an asset.
func Liquidate(a *fixedasset.FixedAsset) error {
	if a.State == fixedasset.StateScrapped || a.State == fixedasset.StateSold {
		return fixedasset.ErrConflict
	}
	a.State = fixedasset.StatePendingLiquidation
	a.UpdatedAt = fixedasset.NowRFC3339()
	return nil
}

// ConfirmLiquidation marks an asset as scrapped or sold.
func ConfirmLiquidation(a *fixedasset.FixedAsset, newState fixedasset.AssetState) error {
	if a.State != fixedasset.StatePendingLiquidation {
		return fixedasset.ErrConflict
	}
	if newState != fixedasset.StateScrapped && newState != fixedasset.StateSold {
		return fixedasset.ErrInvalid
	}
	a.State = newState
	a.UpdatedAt = fixedasset.NowRFC3339()
	return nil
}

// Deactivate moves an active asset to inactive state.
func Deactivate(a *fixedasset.FixedAsset) error {
	if a.State != fixedasset.StateActive {
		return fixedasset.ErrConflict
	}
	a.State = fixedasset.StateInactive
	a.UpdatedAt = fixedasset.NowRFC3339()
	return nil
}

// Reactivate moves an inactive asset back to active state.
func Reactivate(a *fixedasset.FixedAsset) error {
	if a.State != fixedasset.StateInactive {
		return fixedasset.ErrConflict
	}
	a.State = fixedasset.StateActive
	a.UpdatedAt = fixedasset.NowRFC3339()
	return nil
}
