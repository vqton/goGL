package tools

import (
	"context"
	"fmt"

	"goGL/internal/domain/tools"
	"goGL/internal/domain/core"
)

type Service interface {
	Create(ctx context.Context, c *tools.ToolCard, actor string) (*tools.ToolCard, error)
	Get(ctx context.Context, id string) (*tools.ToolCard, error)
	Update(ctx context.Context, id string, patch *tools.ToolCard, actor string) (*tools.ToolCard, error)
	List(ctx context.Context, category string, state tools.CardState, limit, offset int) ([]*tools.ToolCard, error)
	Delete(ctx context.Context, id string) error

	// Transaction operations
	Import(ctx context.Context, toolCardID string, quantity int, unitPrice int64, ref string, actor string) (*tools.ToolTransaction, error)
	Export(ctx context.Context, toolCardID string, quantity int, toDepartment, toPerson string, actor string) (*tools.ToolTransaction, error)
	Transfer(ctx context.Context, toolCardID string, quantity int, toLocation, toDepartment string, actor string) (*tools.ToolTransaction, error)
	Return(ctx context.Context, toolCardID string, quantity int, reason, ref string, actor string) (*tools.ToolTransaction, error)
	Dispose(ctx context.Context, toolCardID string, quantity int, reason string, actor string) (*tools.ToolTransaction, error)

	// Inventory operations
	GetStock(ctx context.Context, toolCardID string) (int, error)
	ListTransactions(ctx context.Context, toolCardID string, txType tools.TransactionType, limit, offset int) ([]*tools.ToolTransaction, error)

	// Scrap (legacy compatibility)
	Scrap(ctx context.Context, id, actor string) (*tools.ToolCard, error)
}

type service struct {
	repo tools.Repository
}

func NewService(repo tools.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, c *tools.ToolCard, actor string) (*tools.ToolCard, error) {
	card := c.Clone()
	card.State = tools.StateActive
	card.CreatedBy = actor
	card.UpdatedBy = actor

	if err := tools.ValidateToolCard(card); err != nil {
		return nil, err
	}

	n, err := s.repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	card.Code = fmt.Sprintf("TL-%05d", n)
	card.ID = core.RowID("tools", card.Code)

	now := core.NowRFC3339()
	card.CreatedAt = now
	card.UpdatedAt = now

	if err := s.repo.Create(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *service) Get(ctx context.Context, id string) (*tools.ToolCard, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id string, patch *tools.ToolCard, actor string) (*tools.ToolCard, error) {
	cur, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.State == tools.StateDisposed {
		return nil, tools.ErrConflict
	}

	if patch.Name != "" {
		cur.Name = patch.Name
	}
	if patch.Category != "" {
		cur.Category = patch.Category
	}
	if patch.SubCategory != "" {
		cur.SubCategory = patch.SubCategory
	}
	if patch.Description != "" {
		cur.Description = patch.Description
	}
	if patch.OriginalCost > 0 {
		cur.OriginalCost = patch.OriginalCost
	}
	if patch.Quantity > 0 {
		cur.Quantity = patch.Quantity
	}
	if patch.Unit != "" {
		cur.Unit = patch.Unit
	}
	if patch.PurchaseDate != "" {
		cur.PurchaseDate = patch.PurchaseDate
	}
	if patch.InvoiceNo != "" {
		cur.InvoiceNo = patch.InvoiceNo
	}
	if patch.Supplier != "" {
		cur.Supplier = patch.Supplier
	}
	if patch.Warehouse != "" {
		cur.Warehouse = patch.Warehouse
	}
	if patch.Location != "" {
		cur.Location = patch.Location
	}
	if patch.Department != "" {
		cur.Department = patch.Department
	}
	if patch.AssignedTo != "" {
		cur.AssignedTo = patch.AssignedTo
	}
	if patch.AccountCode153 != "" {
		cur.AccountCode153 = patch.AccountCode153
	}
	if patch.AccountCodeExp != "" {
		cur.AccountCodeExp = patch.AccountCodeExp
	}

	if err := tools.ValidateToolCard(cur); err != nil {
		return nil, err
	}

	cur.UpdatedBy = actor
	cur.UpdatedAt = core.NowRFC3339()

	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func (s *service) List(ctx context.Context, category string, state tools.CardState, limit, offset int) ([]*tools.ToolCard, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, category, state, limit, offset)
}

func (s *service) Scrap(ctx context.Context, id, actor string) (*tools.ToolCard, error) {
	card, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if card.State == tools.StateDisposed {
		return nil, tools.ErrConflict
	}

	// Delegate to Dispose to create proper transaction record
	_, err = s.Dispose(ctx, id, card.Quantity, "Scrapped", actor)
	if err != nil {
		return nil, err
	}

	return s.repo.FindByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	card, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if card.State != tools.StateActive {
		return tools.ErrConflict
	}
	return s.repo.Delete(ctx, id)
}

// --- Transaction operations ---

func (s *service) Import(ctx context.Context, toolCardID string, quantity int, unitPrice int64, ref string, actor string) (*tools.ToolTransaction, error) {
	if quantity <= 0 {
		return nil, tools.ErrInvalidQuantity
	}

	card, err := s.repo.FindByID(ctx, toolCardID)
	if err != nil {
		return nil, err
	}

	// Atomic stock increment
	if err := s.repo.IncrementStock(ctx, toolCardID, quantity); err != nil {
		return nil, err
	}

	tx := &tools.ToolTransaction{
		ID:              core.RowID("tool_tx", toolCardID),
		ToolCardID:      toolCardID,
		ToolCardCode:    card.Code,
		TransactionType: tools.TxImport,
		Quantity:        quantity,
		UnitPrice:       unitPrice,
		TotalAmount:     int64(quantity) * unitPrice,
		ReferenceNo:     ref,
		GLPosted:        true,
		CreatedBy:       actor,
		CreatedAt:       core.NowRFC3339(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		// Rollback stock on failure
		_ = s.repo.DecrementStock(ctx, toolCardID, quantity)
		return nil, err
	}

	card.Quantity += quantity
	card.UpdatedBy = actor
	card.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *service) Export(ctx context.Context, toolCardID string, quantity int, toDepartment, toPerson string, actor string) (*tools.ToolTransaction, error) {
	if quantity <= 0 {
		return nil, tools.ErrInvalidQuantity
	}

	card, err := s.repo.FindByID(ctx, toolCardID)
	if err != nil {
		return nil, err
	}

	// Atomic stock decrement — prevents race conditions
	if err := s.repo.DecrementStock(ctx, toolCardID, quantity); err != nil {
		return nil, err
	}

	tx := &tools.ToolTransaction{
		ID:              core.RowID("tool_tx", toolCardID),
		ToolCardID:      toolCardID,
		ToolCardCode:    card.Code,
		TransactionType: tools.TxExport,
		Quantity:        quantity,
		UnitPrice:       card.OriginalCost,
		TotalAmount:     int64(quantity) * card.OriginalCost,
		ToDepartment:    toDepartment,
		AssignedTo:      toPerson,
		GLPosted:        true,
		CreatedBy:       actor,
		CreatedAt:       core.NowRFC3339(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		// Rollback stock on transaction creation failure
		_ = s.repo.IncrementStock(ctx, toolCardID, quantity)
		return nil, err
	}

	card.Quantity -= quantity
	card.Department = toDepartment
	card.AssignedTo = toPerson
	card.UpdatedBy = actor
	card.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *service) Transfer(ctx context.Context, toolCardID string, quantity int, toLocation, toDepartment string, actor string) (*tools.ToolTransaction, error) {
	if quantity <= 0 {
		return nil, tools.ErrInvalidQuantity
	}

	card, err := s.repo.FindByID(ctx, toolCardID)
	if err != nil {
		return nil, err
	}

	// Atomic stock decrement
	if err := s.repo.DecrementStock(ctx, toolCardID, quantity); err != nil {
		return nil, err
	}

	tx := &tools.ToolTransaction{
		ID:              core.RowID("tool_tx", toolCardID),
		ToolCardID:      toolCardID,
		ToolCardCode:    card.Code,
		TransactionType: tools.TxTransfer,
		Quantity:        quantity,
		UnitPrice:       card.OriginalCost,
		TotalAmount:     int64(quantity) * card.OriginalCost,
		FromLocation:    card.Location,
		ToLocation:      toLocation,
		FromDepartment:  card.Department,
		ToDepartment:    toDepartment,
		GLPosted:        false,
		CreatedBy:       actor,
		CreatedAt:       core.NowRFC3339(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		_ = s.repo.IncrementStock(ctx, toolCardID, quantity)
		return nil, err
	}

	card.Location = toLocation
	card.Department = toDepartment
	card.UpdatedBy = actor
	card.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *service) Return(ctx context.Context, toolCardID string, quantity int, reason, ref string, actor string) (*tools.ToolTransaction, error) {
	if quantity <= 0 {
		return nil, tools.ErrInvalidQuantity
	}

	card, err := s.repo.FindByID(ctx, toolCardID)
	if err != nil {
		return nil, err
	}

	// Atomic stock decrement
	if err := s.repo.DecrementStock(ctx, toolCardID, quantity); err != nil {
		return nil, err
	}

	tx := &tools.ToolTransaction{
		ID:              core.RowID("tool_tx", toolCardID),
		ToolCardID:      toolCardID,
		ToolCardCode:    card.Code,
		TransactionType: tools.TxReturn,
		Quantity:        quantity,
		UnitPrice:       card.OriginalCost,
		TotalAmount:     int64(quantity) * card.OriginalCost,
		ReferenceNo:     ref,
		Reason:          reason,
		GLPosted:        true,
		CreatedBy:       actor,
		CreatedAt:       core.NowRFC3339(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		_ = s.repo.IncrementStock(ctx, toolCardID, quantity)
		return nil, err
	}

	card.Quantity -= quantity
	card.UpdatedBy = actor
	card.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *service) Dispose(ctx context.Context, toolCardID string, quantity int, reason string, actor string) (*tools.ToolTransaction, error) {
	if quantity <= 0 {
		return nil, tools.ErrInvalidQuantity
	}

	card, err := s.repo.FindByID(ctx, toolCardID)
	if err != nil {
		return nil, err
	}

	// Atomic stock decrement
	if err := s.repo.DecrementStock(ctx, toolCardID, quantity); err != nil {
		return nil, err
	}

	tx := &tools.ToolTransaction{
		ID:              core.RowID("tool_tx", toolCardID),
		ToolCardID:      toolCardID,
		ToolCardCode:    card.Code,
		TransactionType: tools.TxDisposal,
		Quantity:        quantity,
		UnitPrice:       card.OriginalCost,
		TotalAmount:     int64(quantity) * card.OriginalCost,
		Reason:          reason,
		GLPosted:        true,
		CreatedBy:       actor,
		CreatedAt:       core.NowRFC3339(),
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		_ = s.repo.IncrementStock(ctx, toolCardID, quantity)
		return nil, err
	}

	card.Quantity -= quantity
	if card.Quantity <= 0 {
		card.State = tools.StateDisposed
	}
	card.UpdatedBy = actor
	card.UpdatedAt = core.NowRFC3339()
	if err := s.repo.Update(ctx, card); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *service) GetStock(ctx context.Context, toolCardID string) (int, error) {
	return s.repo.GetStock(ctx, toolCardID)
}

func (s *service) ListTransactions(ctx context.Context, toolCardID string, txType tools.TransactionType, limit, offset int) ([]*tools.ToolTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListTransactions(ctx, toolCardID, txType, limit, offset)
}
