package bank

import (
	"context"
	"testing"

	"goGL/internal/domain/bank"
)

func TestCreateTransaction_Success(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{}}
	svc := NewService(repo)

	input := &bank.BankTransaction{
		AccountNo: "1234567890",
		BankCode:  "VCB",
		RefDate:   "2026-01-15",
		Amount:    50000000,
		Type:      bank.TxTypeDeposit,
	}

	result, err := svc.Create(context.Background(), input, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code == "" {
		t.Error("expected auto-generated code")
	}
	if result.State != bank.StatePending {
		t.Errorf("state = %q, want pending", result.State)
	}
	if result.Currency != "VND" {
		t.Errorf("currency = %q, want VND", result.Currency)
	}
}

func TestCreateTransaction_EmptyAccount(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{}}
	svc := NewService(repo)

	input := &bank.BankTransaction{
		BankCode: "VCB",
		RefDate:  "2026-01-15",
		Amount:   1000,
		Type:     bank.TxTypeDeposit,
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for empty account number")
	}
}

func TestCreateTransaction_ZeroAmount(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{}}
	svc := NewService(repo)

	input := &bank.BankTransaction{
		AccountNo: "1234567890",
		BankCode:  "VCB",
		RefDate:   "2026-01-15",
		Type:      bank.TxTypeDeposit,
	}

	_, err := svc.Create(context.Background(), input, "admin")
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestGetTransaction_Success(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", AccountNo: "123"},
	}}
	svc := NewService(repo)

	result, err := svc.Get(context.Background(), "tx-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccountNo != "123" {
		t.Errorf("account_no = %q, want 123", result.AccountNo)
	}
}

func TestGetTransaction_NotFound(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{}}
	svc := NewService(repo)

	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent transaction")
	}
}

func TestUpdateTransaction_Pending(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", AccountNo: "123", BankCode: "VCB", RefDate: "2026-01-15", Amount: 1000, Type: bank.TxTypeDeposit, State: bank.StatePending},
	}}
	svc := NewService(repo)

	patch := &bank.BankTransaction{Description: "updated desc"}
	result, err := svc.Update(context.Background(), "tx-1", patch, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Description != "updated desc" {
		t.Errorf("description = %q, want updated desc", result.Description)
	}
}

func TestUpdateTransaction_Cleared(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StateCleared},
	}}
	svc := NewService(repo)

	patch := &bank.BankTransaction{Description: "updated desc"}
	_, err := svc.Update(context.Background(), "tx-1", patch, "admin")
	if err == nil {
		t.Error("expected error for cleared transaction")
	}
}

func TestClearTransaction_Success(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StatePending},
	}}
	svc := NewService(repo)

	result, err := svc.Clear(context.Background(), "tx-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != bank.StateCleared {
		t.Errorf("state = %q, want cleared", result.State)
	}
}

func TestClearTransaction_WrongState(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StateCleared},
	}}
	svc := NewService(repo)

	_, err := svc.Clear(context.Background(), "tx-1", "admin")
	if err == nil {
		t.Error("expected error for already cleared transaction")
	}
}

func TestReconcileTransaction_Success(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StateCleared},
	}}
	svc := NewService(repo)

	result, err := svc.Reconcile(context.Background(), "tx-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != bank.StateReconciled {
		t.Errorf("state = %q, want reconciled", result.State)
	}
}

func TestReconcileTransaction_WrongState(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StatePending},
	}}
	svc := NewService(repo)

	_, err := svc.Reconcile(context.Background(), "tx-1", "admin")
	if err == nil {
		t.Error("expected error for pending transaction")
	}
}

func TestCancelTransaction_Success(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StatePending},
	}}
	svc := NewService(repo)

	result, err := svc.Cancel(context.Background(), "tx-1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.State != bank.StateCancelled {
		t.Errorf("state = %q, want cancelled", result.State)
	}
}

func TestCancelTransaction_Reconciled(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StateReconciled},
	}}
	svc := NewService(repo)

	_, err := svc.Cancel(context.Background(), "tx-1", "admin")
	if err == nil {
		t.Error("expected error for reconciled transaction")
	}
}

func TestDeleteTransaction_Pending(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StatePending},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "tx-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteTransaction_Cleared(t *testing.T) {
	repo := &mockRepo{txs: map[string]*bank.BankTransaction{
		"tx-1": {ID: "tx-1", State: bank.StateCleared},
	}}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), "tx-1")
	if err == nil {
		t.Error("expected error for cleared transaction")
	}
}

func TestTransactionType_IsValid(t *testing.T) {
	tests := []struct {
		typ  bank.TransactionType
		want bool
	}{
		{bank.TxTypeDeposit, true},
		{bank.TxTypeWithdrawal, true},
		{bank.TxTypeTransfer, true},
		{bank.TxTypeFee, true},
		{bank.TxTypeInterest, true},
		{bank.TxTypeOther, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.typ.IsValid(); got != tt.want {
			t.Errorf("TransactionType(%q).IsValid() = %v, want %v", tt.typ, got, tt.want)
		}
	}
}

// mockRepo is an in-memory repository for testing.
type mockRepo struct {
	txs map[string]*bank.BankTransaction
	seq int64
}

func (m *mockRepo) Create(_ context.Context, t *bank.BankTransaction) error {
	m.txs[t.ID] = t
	return nil
}

func (m *mockRepo) FindByID(_ context.Context, id string) (*bank.BankTransaction, error) {
	if t, ok := m.txs[id]; ok {
		return t, nil
	}
	return nil, bank.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, t *bank.BankTransaction) error {
	m.txs[t.ID] = t
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	delete(m.txs, id)
	return nil
}

func (m *mockRepo) List(_ context.Context, accountNo string, txType bank.TransactionType) ([]*bank.BankTransaction, error) {
	var out []*bank.BankTransaction
	for _, t := range m.txs {
		if accountNo != "" && t.AccountNo != accountNo {
			continue
		}
		if txType != "" && t.Type != txType {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (m *mockRepo) NextCode(_ context.Context) (int64, error) {
	m.seq++
	return m.seq, nil
}
