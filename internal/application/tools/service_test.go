package tools

import (
	"context"
	"testing"

	"goGL/internal/domain/tools"
)

func TestCreateCard_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	input := &tools.ToolCard{
		Name:          "Drill Machine",
		Category:      "tools",
		OriginalCost:  2500000, // 2.5M VND (< 30M threshold)
		Quantity:      2,
		Unit:          "pcs",
		PurchaseDate:  "2026-08-29",
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != tools.StateActive {
		t.Errorf("state = %q, want active", result.State)
	}
}

func TestCreateCard_EmptyName(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	input := &tools.ToolCard{
		Category: "power",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateCard_NegativePrice(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	input := &tools.ToolCard{
		Name:         "Test",
		OriginalCost: -1,
		PurchaseDate: "2026-08-29",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for negative price")
	}
}

func TestCreateCard_CodeFormat(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	result, _ := svc.Create(context.Background(), &tools.ToolCard{
		Name:         "Test",
		OriginalCost: 1000000,
		PurchaseDate: "2026-08-29",
	}, "admin")
	if len(result.Code) != 8 || result.Code[:3] != "TL-" {
		t.Errorf("code = %q, want TL-XXXXX format", result.Code)
	}
}

func TestGetCard_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Name: "Test", OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	result, err := svc.Get(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Test" {
		t.Errorf("name = %q, want Test", result.Name)
	}
}

func TestGetCard_NotFound(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent card")
	}
}

func TestUpdateCard_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Name: "Old", State: tools.StateActive, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	patch := &tools.ToolCard{Name: "New Name"}
	result, err := svc.Update(context.Background(), "card-1", patch, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "New Name" {
		t.Errorf("name = %q, want New Name", result.Name)
	}
}

func TestUpdateCard_Disposed(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", State: tools.StateDisposed, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	patch := &tools.ToolCard{Name: "New Name"}
	_, err := svc.Update(context.Background(), "card-1", patch, "admin")
	if err == nil {
		t.Error("expected error for disposed card")
	}
}

func TestScrapCard_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", State: tools.StateActive, OriginalCost: 1000000, Quantity: 5, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	result, err := svc.Scrap(context.Background(), "card-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != tools.StateDisposed {
		t.Errorf("state = %q, want disposed", result.State)
	}
}

func TestScrapCard_AlreadyDisposed(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", State: tools.StateDisposed, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	_, err := svc.Scrap(context.Background(), "card-1", "admin")
	if err == nil {
		t.Error("expected error for already disposed card")
	}
}

func TestDeleteCard_Active(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", State: tools.StateActive, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCard_Disposed(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", State: tools.StateDisposed, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "card-1")
	if err == nil {
		t.Error("expected error for disposed card")
	}
}

func TestCardState_IsValid(t *testing.T) {
	tests := []struct {
		s    tools.CardState
		want bool
	}{
		{tools.StateActive, true},
		{tools.StateInactive, true},
		{tools.StateDisposed, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.s.IsValid(); got != tt.want {
			t.Errorf("CardState(%q).IsValid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestCreateCard_OverThreshold(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	// Item >= 30M VND should be rejected (use fixedasset module)
	input := &tools.ToolCard{
		Name:         "Expensive Equipment",
		Category:     "tools",
		OriginalCost: 30_000_000, // At threshold
		Quantity:     1,
		PurchaseDate: "2026-08-29",
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for item >= 30M VND")
	}
}

func TestCreateCard_UnderThreshold(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	// Item < 30M VND should be accepted
	input := &tools.ToolCard{
		Name:         "Drill Machine",
		Category:     "tools",
		OriginalCost: 29_999_999, // Just under threshold
		Quantity:     1,
		PurchaseDate: "2026-08-29",
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
}

func TestCreateCard_MissingPurchaseDate(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{}}
	svc := NewService(repo)

	input := &tools.ToolCard{
		Name:         "Drill Machine",
		Category:     "tools",
		OriginalCost: 2500000,
		Quantity:     1,
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for missing purchase date")
	}
}

// --- Transaction tests ---

func TestImport_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 5, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	tx, err := svc.Import(context.Background(), "card-1", 10, 1000000, "INV-001", "admin")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if tx.TransactionType != tools.TxImport {
		t.Errorf("type = %q, want import", tx.TransactionType)
	}
	if tx.Quantity != 10 {
		t.Errorf("quantity = %d, want 10", tx.Quantity)
	}
	if tx.TotalAmount != 10000000 {
		t.Errorf("total = %d, want 10000000", tx.TotalAmount)
	}
}

func TestImport_InvalidQuantity(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 5, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	_, err := svc.Import(context.Background(), "card-1", 0, 1000000, "INV-001", "admin")
	if err != tools.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity, got %v", err)
	}
}

func TestExport_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 10, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	tx, err := svc.Export(context.Background(), "card-1", 5, "production", "user1", "admin")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if tx.TransactionType != tools.TxExport {
		t.Errorf("type = %q, want export", tx.TransactionType)
	}
	if tx.Quantity != 5 {
		t.Errorf("quantity = %d, want 5", tx.Quantity)
	}
}

func TestExport_InsufficientStock(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 5, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	_, err := svc.Export(context.Background(), "card-1", 10, "production", "user1", "admin")
	if err != tools.ErrInsufficientStock {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestTransfer_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 10, Location: "WH-01", Department: "production", OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	tx, err := svc.Transfer(context.Background(), "card-1", 5, "WH-02", "management", "admin")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if tx.TransactionType != tools.TxTransfer {
		t.Errorf("type = %q, want transfer", tx.TransactionType)
	}
}

func TestReturn_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 10, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	tx, err := svc.Return(context.Background(), "card-1", 3, "Defective", "INV-001", "admin")
	if err != nil {
		t.Fatalf("Return: %v", err)
	}
	if tx.TransactionType != tools.TxReturn {
		t.Errorf("type = %q, want return", tx.TransactionType)
	}
}

func TestDispose_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 5, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	tx, err := svc.Dispose(context.Background(), "card-1", 5, "Old and broken", "admin")
	if err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	if tx.TransactionType != tools.TxDisposal {
		t.Errorf("type = %q, want disposal", tx.TransactionType)
	}
}

func TestGetStock_Success(t *testing.T) {
	repo := &mockRepo{cards: map[string]*tools.ToolCard{
		"card-1": {ID: "card-1", Code: "TL-00001", Name: "Test", Quantity: 10, OriginalCost: 1000000, PurchaseDate: "2026-08-29"},
	}}
	svc := NewService(repo)

	stock, err := svc.GetStock(context.Background(), "card-1")
	if err != nil {
		t.Fatalf("GetStock: %v", err)
	}
	if stock != 10 {
		t.Errorf("stock = %d, want 10", stock)
	}
}

// mockRepo is an in-memory repository for testing.
type mockRepo struct {
	cards map[string]*tools.ToolCard
	seq   int64
}

func (m *mockRepo) Create(_ context.Context, c *tools.ToolCard) error {
	m.cards[c.ID] = c
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id string) (*tools.ToolCard, error) {
	if c, ok := m.cards[id]; ok {
		return c, nil
	}
	return nil, tools.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, c *tools.ToolCard) error {
	m.cards[c.ID] = c
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	delete(m.cards, id)
	return nil
}

func (m *mockRepo) List(_ context.Context, category string, state tools.CardState, limit, offset int) ([]*tools.ToolCard, error) {
	var out []*tools.ToolCard
	for _, c := range m.cards {
		if category != "" && c.Category != category {
			continue
		}
		if state != "" && c.State != state {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (m *mockRepo) NextCode(_ context.Context) (int64, error) {
	m.seq++
	return m.seq, nil
}

func (m *mockRepo) CreateTransaction(_ context.Context, tx *tools.ToolTransaction) error {
	return nil
}

func (m *mockRepo) FindTransactionByID(_ context.Context, id string) (*tools.ToolTransaction, error) {
	return nil, tools.ErrNotFound
}

func (m *mockRepo) ListTransactions(_ context.Context, toolCardID string, txType tools.TransactionType, limit, offset int) ([]*tools.ToolTransaction, error) {
	return nil, nil
}

func (m *mockRepo) GetStock(_ context.Context, toolCardID string) (int, error) {
	if c, ok := m.cards[toolCardID]; ok {
		return c.Quantity, nil
	}
	return 0, nil
}

func (m *mockRepo) AdjustStock(_ context.Context, toolCardID string, quantity int) error {
	return nil
}

func (m *mockRepo) DecrementStock(_ context.Context, toolCardID string, quantity int) error {
	if c, ok := m.cards[toolCardID]; ok {
		if c.Quantity < quantity {
			return tools.ErrInsufficientStock
		}
		c.Quantity -= quantity
		return nil
	}
	return tools.ErrNotFound
}

func (m *mockRepo) IncrementStock(_ context.Context, toolCardID string, quantity int) error {
	if c, ok := m.cards[toolCardID]; ok {
		c.Quantity += quantity
		return nil
	}
	return tools.ErrNotFound
}
