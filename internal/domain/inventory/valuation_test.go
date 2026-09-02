package inventory

import "testing"

func TestFIFOAlloc_SingleLayer(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 100, UnitCost: 50000},
	}
	results, totalCost, err := FIFOAlloc(layers, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Qty != 50 {
		t.Errorf("expected qty 50, got %.2f", results[0].Qty)
	}
	if results[0].Cost != 2500000 {
		t.Errorf("expected cost 2500000, got %d", results[0].Cost)
	}
	if totalCost != 2500000 {
		t.Errorf("expected total 2500000, got %d", totalCost)
	}
}

func TestFIFOAlloc_MultipleLayers(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 30, UnitCost: 50000},
		{ID: "L2", RemainingQty: 70, UnitCost: 55000},
		{ID: "L3", RemainingQty: 100, UnitCost: 60000},
	}
	results, totalCost, err := FIFOAlloc(layers, 80)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// L1: 30 @ 50000 = 1,500,000
	if results[0].LayerID != "L1" || results[0].Qty != 30 || results[0].Cost != 1500000 {
		t.Errorf("L1: got id=%s qty=%.2f cost=%d", results[0].LayerID, results[0].Qty, results[0].Cost)
	}
	// L2: 50 @ 55000 = 2,750,000
	if results[1].LayerID != "L2" || results[1].Qty != 50 || results[1].Cost != 2750000 {
		t.Errorf("L2: got id=%s qty=%.2f cost=%d", results[1].LayerID, results[1].Qty, results[1].Cost)
	}
	// Total: 1,500,000 + 2,750,000 = 4,250,000
	if totalCost != 4250000 {
		t.Errorf("expected total 4250000, got %d", totalCost)
	}
}

func TestFIFOAlloc_ExactlyAllLayers(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 10, UnitCost: 100},
		{ID: "L2", RemainingQty: 20, UnitCost: 200},
	}
	results, totalCost, err := FIFOAlloc(layers, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if totalCost != 5000 {
		t.Errorf("expected total 5000, got %d", totalCost)
	}
}

func TestFIFOAlloc_InsufficientStock(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 10, UnitCost: 50000},
	}
	_, _, err := FIFOAlloc(layers, 20)
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}
}

func TestFIFOAlloc_ZeroQty(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 10, UnitCost: 50000},
	}
	_, _, err := FIFOAlloc(layers, 0)
	if err == nil {
		t.Fatal("expected error for zero qty")
	}
}

func TestFIFOAlloc_SkipsExhaustedLayers(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 0, UnitCost: 50000},
		{ID: "L2", RemainingQty: 50, UnitCost: 55000},
	}
	results, totalCost, err := FIFOAlloc(layers, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].LayerID != "L2" {
		t.Errorf("expected L2, got %s", results[0].LayerID)
	}
	if totalCost != 1650000 {
		t.Errorf("expected 1650000, got %d", totalCost)
	}
}

func TestFIFOAlloc_PartialLastLayer(t *testing.T) {
	layers := []*StockValuationLayer{
		{ID: "L1", RemainingQty: 100, UnitCost: 50000},
	}
	results, totalCost, err := FIFOAlloc(layers, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Qty != 100 || totalCost != 5000000 {
		t.Errorf("expected qty=100 cost=5000000, got qty=%.2f cost=%d", results[0].Qty, totalCost)
	}
}

func TestWeightedAvgCost_NewItem(t *testing.T) {
	// New item: no prior stock
	avg := WeightedAvgCost(0, 0, 100, 50000)
	if avg != 50000 {
		t.Errorf("expected 50000, got %d", avg)
	}
}

func TestWeightedAvgCost_AddToExisting(t *testing.T) {
	// Existing: 100 @ 50000, receive 50 @ 60000
	// New avg: (100*50000 + 50*60000) / 150 = 8000000/150 = 53333
	avg := WeightedAvgCost(100, 50000, 50, 60000)
	if avg != 53333 {
		t.Errorf("expected 53333, got %d", avg)
	}
}

func TestWeightedAvgCost_SamePrice(t *testing.T) {
	avg := WeightedAvgCost(100, 50000, 50, 50000)
	if avg != 50000 {
		t.Errorf("expected 50000, got %d", avg)
	}
}

func TestStockValuationLayer_Clone(t *testing.T) {
	l := &StockValuationLayer{ID: "1", ItemCode: "MH-001", RemainingQty: 10}
	cp := l.Clone()
	cp.RemainingQty = 5
	if l.RemainingQty == cp.RemainingQty {
		t.Error("clone should not share memory")
	}
}

func TestStockValuationLayer_DispatchQty(t *testing.T) {
	l := &StockValuationLayer{RemainingQty: 50}

	// Request less than available
	if got := l.DispatchQty(30); got != 30 {
		t.Errorf("expected 30, got %.2f", got)
	}

	// Request more than available
	if got := l.DispatchQty(100); got != 50 {
		t.Errorf("expected 50, got %.2f", got)
	}

	// Request exactly available
	if got := l.DispatchQty(50); got != 50 {
		t.Errorf("expected 50, got %.2f", got)
	}

	// Exhausted layer
	l.RemainingQty = 0
	if got := l.DispatchQty(10); got != 0 {
		t.Errorf("expected 0, got %.2f", got)
	}
}
