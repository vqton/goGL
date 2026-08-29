package bank

import (
	"context"
	"fmt"

	"goGL/internal/domain/bank"
	"goGL/internal/domain/core"
)

type Service interface {
	Create(ctx context.Context, t *bank.BankTransaction, actor string) (*bank.BankTransaction, error)
	Get(ctx context.Context, id string) (*bank.BankTransaction, error)
	Update(ctx context.Context, id string, patch *bank.BankTransaction, actor string) (*bank.BankTransaction, error)
	List(ctx context.Context, accountNo string, txType bank.TransactionType) ([]*bank.BankTransaction, error)
	Clear(ctx context.Context, id, actor string) (*bank.BankTransaction, error)
	Reconcile(ctx context.Context, id, actor string) (*bank.BankTransaction, error)
	Cancel(ctx context.Context, id, actor string) (*bank.BankTransaction, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo bank.Repository
}

func NewService(repo bank.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, t *bank.BankTransaction, actor string) (*bank.BankTransaction, error) {
	tx := t.Clone()
	tx.State = bank.StatePending
	tx.CreatedBy = actor
	tx.UpdatedBy = actor

	if err := bank.ValidateTransaction(tx); err != nil {
		return nil, err
	}

	n, err := s.repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	tx.Code = fmt.Sprintf("BANK-%05d", n)
	tx.ID = core.RowID("bank", tx.Code)

	now := core.NowRFC3339()
	tx.CreatedAt = now
	tx.UpdatedAt = now

	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *service) Get(ctx context.Context, id string) (*bank.BankTransaction, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, patch *bank.BankTransaction, actor string) (*bank.BankTransaction, error) {
	cur, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.State != bank.StatePending {
		return nil, bank.ErrConflict
	}

	if patch.AccountNo != "" {
		cur.AccountNo = patch.AccountNo
	}
	if patch.BankCode != "" {
		cur.BankCode = patch.BankCode
	}
	if patch.BankName != "" {
		cur.BankName = patch.BankName
	}
	if patch.RefDate != "" {
		cur.RefDate = patch.RefDate
	}
	if patch.ValueDate != "" {
		cur.ValueDate = patch.ValueDate
	}
	if patch.Amount != 0 {
		cur.Amount = patch.Amount
	}
	if patch.Currency != "" {
		cur.Currency = patch.Currency
	}
	if patch.Type != "" {
		cur.Type = patch.Type
	}
	if patch.Description != "" {
		cur.Description = patch.Description
	}
	if patch.Counterparty != "" {
		cur.Counterparty = patch.Counterparty
	}
	if patch.Reference != "" {
		cur.Reference = patch.Reference
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}

	if err := bank.ValidateTransaction(cur); err != nil {
		return nil, err
	}

	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) List(ctx context.Context, accountNo string, txType bank.TransactionType) ([]*bank.BankTransaction, error) {
	return s.repo.List(ctx, accountNo, txType)
}

func (s *service) Clear(ctx context.Context, id, actor string) (*bank.BankTransaction, error) {
	tx, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tx.State != bank.StatePending {
		return nil, bank.ErrConflict
	}
	tx.State = bank.StateCleared
	tx.UpdatedBy = actor
	tx.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *service) Reconcile(ctx context.Context, id, actor string) (*bank.BankTransaction, error) {
	tx, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tx.State != bank.StateCleared {
		return nil, bank.ErrConflict
	}
	tx.State = bank.StateReconciled
	tx.UpdatedBy = actor
	tx.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *service) Cancel(ctx context.Context, id, actor string) (*bank.BankTransaction, error) {
	tx, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tx.State == bank.StateCancelled || tx.State == bank.StateReconciled {
		return nil, bank.ErrConflict
	}
	tx.State = bank.StateCancelled
	tx.UpdatedBy = actor
	tx.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	tx, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tx.State != bank.StatePending {
		return bank.ErrConflict
	}
	return s.repo.Delete(ctx, id)
}
