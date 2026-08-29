package contract

import (
	"context"
	"fmt"

	"goGL/internal/domain/contract"
	"goGL/internal/domain/core"
)

type Service interface {
	Create(ctx context.Context, c *contract.Contract, actor string) (*contract.Contract, error)
	Get(ctx context.Context, id string) (*contract.Contract, error)
	Update(ctx context.Context, id string, patch *contract.Contract, actor string) (*contract.Contract, error)
	List(ctx context.Context, ctype contract.ContractType, state contract.ContractState) ([]*contract.Contract, error)
	Activate(ctx context.Context, id, actor string) (*contract.Contract, error)
	Terminate(ctx context.Context, id, actor string) (*contract.Contract, error)
	Delete(ctx context.Context, id string) error
	CreateLoan(ctx context.Context, l *contract.LoanAgreement, actor string) (*contract.LoanAgreement, error)
	GetLoan(ctx context.Context, id string) (*contract.LoanAgreement, error)
}

type service struct {
	repo contract.Repository
}

func NewService(repo contract.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, c *contract.Contract, actor string) (*contract.Contract, error) {
	c = c.Clone()
	c.State = contract.StateDraft
	c.CreatedBy = actor
	c.UpdatedBy = actor

	if err := contract.ValidateContract(c); err != nil {
		return nil, err
	}

	n, err := s.repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	c.Code = fmt.Sprintf("CTR-%05d", n)
	c.ID = core.RowID("contract", c.Code)

	now := core.NowRFC3339()
	c.CreatedAt = now
	c.UpdatedAt = now

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *service) Get(ctx context.Context, id string) (*contract.Contract, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, patch *contract.Contract, actor string) (*contract.Contract, error) {
	cur, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.State != contract.StateDraft {
		return nil, contract.ErrLocked
	}

	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Type != "" {
		cur.Type = patch.Type
	}
	if patch.PartyName != "" {
		cur.PartyName = patch.PartyName
	}
	if patch.PartyTaxID != "" {
		cur.PartyTaxID = patch.PartyTaxID
	}
	if patch.StartDate != "" {
		cur.StartDate = patch.StartDate
	}
	if patch.EndDate != "" {
		cur.EndDate = patch.EndDate
	}
	if patch.Value > 0 {
		cur.Value = patch.Value
	}
	if patch.Currency != "" {
		cur.Currency = patch.Currency
	}
	if patch.Notes != "" {
		cur.Notes = patch.Notes
	}

	if err := contract.ValidateContract(cur); err != nil {
		return nil, err
	}

	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) List(ctx context.Context, ctype contract.ContractType, state contract.ContractState) ([]*contract.Contract, error) {
	return s.repo.List(ctx, ctype, state)
}

func (s *service) Activate(ctx context.Context, id, actor string) (*contract.Contract, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.State != contract.StateDraft {
		return nil, contract.ErrLocked
	}

	now := core.NowRFC3339()
	if c.EndDate < now {
		return nil, &core.ValidationError{Field: "end_date", Message: "contract has already expired"}
	}
	if c.StartDate > now {
		return nil, &core.ValidationError{Field: "start_date", Message: "contract start date is in the future"}
	}

	c.State = contract.StateActive
	c.UpdatedBy = actor
	c.UpdatedAt = now
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *service) Terminate(ctx context.Context, id, actor string) (*contract.Contract, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.State != contract.StateActive {
		return nil, contract.ErrLocked
	}
	c.State = contract.StateTerminated
	c.UpdatedBy = actor
	c.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if c.State != contract.StateDraft {
		return contract.ErrLocked
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) CreateLoan(ctx context.Context, l *contract.LoanAgreement, actor string) (*contract.LoanAgreement, error) {
	n, err := s.repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	l.Code = fmt.Sprintf("LN-%05d", n)
	l.ID = core.RowID("loan", l.Code)

	now := core.NowRFC3339()
	l.CreatedAt = now
	l.UpdatedAt = now
	l.Status = "active"

	if err := s.repo.CreateLoan(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *service) GetLoan(ctx context.Context, id string) (*contract.LoanAgreement, error) {
	return s.repo.FindLoanByID(ctx, id)
}
