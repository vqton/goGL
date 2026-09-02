package inventory

import "fmt"

// DispatchResult tracks the cost allocation for a single dispatch.
type DispatchResult struct {
	LayerID string
	Qty     float64
	Cost    int64
}

// FIFOAlloc dispatches qty using FIFO (first-in, first-out).
// Returns the cost allocation per layer and total cost.
// Layers must be ordered by received_date ASC (oldest first).
func FIFOAlloc(layers []*StockValuationLayer, qty float64) ([]DispatchResult, int64, error) {
	if qty <= 0 {
		return nil, 0, fmt.Errorf("inventory: fifo alloc: qty must be positive")
	}
	totalAvail := 0.0
	for _, l := range layers {
		totalAvail += l.RemainingQty
	}
	if totalAvail < qty {
		return nil, 0, fmt.Errorf("inventory: fifo alloc: insufficient stock (have %.2f, need %.2f)", totalAvail, qty)
	}

	var results []DispatchResult
	remaining := qty
	var totalCost int64

	for _, l := range layers {
		if remaining <= 0 {
			break
		}
		if l.RemainingQty <= 0 {
			continue
		}
		take := l.DispatchQty(remaining)
		cost := int64(take) * l.UnitCost
		results = append(results, DispatchResult{
			LayerID: l.ID,
			Qty:     take,
			Cost:    cost,
		})
		totalCost += cost
		remaining -= take
	}

	return results, totalCost, nil
}

// WeightedAvgCost returns the new weighted average unit cost after a receipt.
func WeightedAvgCost(oldQty float64, oldAvgCost int64, recvQty float64, recvUnitCost int64) int64 {
	newQty := oldQty + recvQty
	if newQty == 0 {
		return recvUnitCost
	}
	newValue := oldQty*float64(oldAvgCost) + recvQty*float64(recvUnitCost)
	return int64(newValue / newQty)
}
