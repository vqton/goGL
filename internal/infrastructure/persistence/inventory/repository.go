package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goGL/internal/domain/inventory"
)

type sqliteRepository struct {
	db *sql.DB
}

func NewSqliteRepository(db *sql.DB) inventory.Repository {
	return &sqliteRepository{db: db}
}

// --- Sequence helpers ---

func (r *sqliteRepository) nextSeq(ctx context.Context, seqName string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("inventory: seq begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO inventory_sequences (id, data) VALUES (?, '{"seq":0}')
		 ON CONFLICT(id) DO NOTHING`, seqName); err != nil {
		return 0, fmt.Errorf("inventory: seq insert: %w", err)
	}
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE inventory_sequences SET data = json_set(data, '$.seq', json_extract(data, '$.seq') + 1)
		 WHERE id = ? RETURNING json_extract(data, '$.seq')`, seqName).Scan(&seq); err != nil {
		return 0, fmt.Errorf("inventory: seq bump: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("inventory: seq commit: %w", err)
	}
	return seq, nil
}

// --- Item CRUD ---

func (r *sqliteRepository) CreateItem(ctx context.Context, i *inventory.Item) error {
	data, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("inventory: item marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO inventory_items (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		i.ID, string(data))
	if err != nil {
		return fmt.Errorf("inventory: item create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindItemByID(ctx context.Context, id string) (*inventory.Item, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_items WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: item find: %w", err)
	}
	var i inventory.Item
	if err := json.Unmarshal([]byte(data), &i); err != nil {
		return nil, fmt.Errorf("inventory: item decode: %w", err)
	}
	return &i, nil
}

func (r *sqliteRepository) FindItemByCode(ctx context.Context, code string) (*inventory.Item, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_items WHERE json_extract(data, '$.code') = ?`, code).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: item find by code: %w", err)
	}
	var i inventory.Item
	if err := json.Unmarshal([]byte(data), &i); err != nil {
		return nil, fmt.Errorf("inventory: item decode: %w", err)
	}
	return &i, nil
}

func (r *sqliteRepository) UpdateItem(ctx context.Context, i *inventory.Item) error {
	data, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("inventory: item marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE inventory_items SET data = ? WHERE id = ?`,
		string(data), i.ID)
	if err != nil {
		return fmt.Errorf("inventory: item update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteItem(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM inventory_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("inventory: item delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListItems(ctx context.Context, category inventory.ItemCategory, status inventory.ItemStatus, search string, limit, offset int) ([]*inventory.Item, int, error) {
	where := "1=1"
	args := []interface{}{}
	if category != "" {
		where += " AND json_extract(data, '$.category') = ?"
		args = append(args, category)
	}
	if status != "" {
		where += " AND json_extract(data, '$.status') = ?"
		args = append(args, status)
	}
	if search != "" {
		where += " AND (json_extract(data, '$.name') LIKE ? OR json_extract(data, '$.code') LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_items WHERE `+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: item count: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM inventory_items WHERE `+where+` ORDER BY json_extract(data, '$.code') LIMIT ? OFFSET ?`,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: item list: %w", err)
	}
	defer rows.Close()

	var items []*inventory.Item
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, 0, fmt.Errorf("inventory: item scan: %w", err)
		}
		var i inventory.Item
		if err := json.Unmarshal([]byte(data), &i); err != nil {
			return nil, 0, fmt.Errorf("inventory: item decode: %w", err)
		}
		items = append(items, &i)
	}
	return items, total, nil
}

func (r *sqliteRepository) NextItemCode(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "item_code")
}

// --- Warehouse CRUD ---

func (r *sqliteRepository) CreateWarehouse(ctx context.Context, w *inventory.Warehouse) error {
	data, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("inventory: warehouse marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO inventory_warehouses (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		w.ID, string(data))
	if err != nil {
		return fmt.Errorf("inventory: warehouse create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindWarehouseByID(ctx context.Context, id string) (*inventory.Warehouse, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_warehouses WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: warehouse find: %w", err)
	}
	var w inventory.Warehouse
	if err := json.Unmarshal([]byte(data), &w); err != nil {
		return nil, fmt.Errorf("inventory: warehouse decode: %w", err)
	}
	return &w, nil
}

func (r *sqliteRepository) FindWarehouseByCode(ctx context.Context, code string) (*inventory.Warehouse, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_warehouses WHERE json_extract(data, '$.code') = ?`, code).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: warehouse find by code: %w", err)
	}
	var w inventory.Warehouse
	if err := json.Unmarshal([]byte(data), &w); err != nil {
		return nil, fmt.Errorf("inventory: warehouse decode: %w", err)
	}
	return &w, nil
}

func (r *sqliteRepository) UpdateWarehouse(ctx context.Context, w *inventory.Warehouse) error {
	data, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("inventory: warehouse marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE inventory_warehouses SET data = ? WHERE id = ?`,
		string(data), w.ID)
	if err != nil {
		return fmt.Errorf("inventory: warehouse update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) DeleteWarehouse(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM inventory_warehouses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("inventory: warehouse delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListWarehouses(ctx context.Context, status inventory.WarehouseStatus, limit, offset int) ([]*inventory.Warehouse, int, error) {
	where := "1=1"
	args := []interface{}{}
	if status != "" {
		where += " AND json_extract(data, '$.status') = ?"
		args = append(args, status)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_warehouses WHERE `+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: warehouse count: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM inventory_warehouses WHERE `+where+` ORDER BY json_extract(data, '$.code') LIMIT ? OFFSET ?`,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: warehouse list: %w", err)
	}
	defer rows.Close()

	var warehouses []*inventory.Warehouse
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, 0, fmt.Errorf("inventory: warehouse scan: %w", err)
		}
		var w inventory.Warehouse
		if err := json.Unmarshal([]byte(data), &w); err != nil {
			return nil, 0, fmt.Errorf("inventory: warehouse decode: %w", err)
		}
		warehouses = append(warehouses, &w)
	}
	return warehouses, total, nil
}

func (r *sqliteRepository) NextWarehouseCode(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "warehouse_code")
}

// --- Stock Card operations ---

func (r *sqliteRepository) GetStockCard(ctx context.Context, itemCode, warehouseCode string) (*inventory.StockCard, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_stock_cards
		 WHERE json_extract(data, '$.item_code') = ? AND json_extract(data, '$.warehouse_code') = ?`,
		itemCode, warehouseCode).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: stock card find: %w", err)
	}
	var sc inventory.StockCard
	if err := json.Unmarshal([]byte(data), &sc); err != nil {
		return nil, fmt.Errorf("inventory: stock card decode: %w", err)
	}
	return &sc, nil
}

func (r *sqliteRepository) ListStockCards(ctx context.Context, warehouseCode string, limit, offset int) ([]*inventory.StockCard, int, error) {
	where := "1=1"
	args := []interface{}{}
	if warehouseCode != "" {
		where += " AND json_extract(data, '$.warehouse_code') = ?"
		args = append(args, warehouseCode)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_stock_cards WHERE `+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: stock card count: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM inventory_stock_cards WHERE `+where+` ORDER BY json_extract(data, '$.item_code') LIMIT ? OFFSET ?`,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: stock card list: %w", err)
	}
	defer rows.Close()

	var cards []*inventory.StockCard
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, 0, fmt.Errorf("inventory: stock card scan: %w", err)
		}
		var sc inventory.StockCard
		if err := json.Unmarshal([]byte(data), &sc); err != nil {
			return nil, 0, fmt.Errorf("inventory: stock card decode: %w", err)
		}
		cards = append(cards, &sc)
	}
	return cards, total, nil
}

func (r *sqliteRepository) UpsertStockCard(ctx context.Context, sc *inventory.StockCard) error {
	data, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("inventory: stock card marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO inventory_stock_cards (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		sc.ID, string(data))
	if err != nil {
		return fmt.Errorf("inventory: stock card upsert: %w", err)
	}
	return nil
}

// --- Stock Movement operations ---

func (r *sqliteRepository) CreateMovement(ctx context.Context, m *inventory.StockMovement) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("inventory: movement marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO inventory_stock_movements (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		m.ID, string(data))
	if err != nil {
		return fmt.Errorf("inventory: movement create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindMovementByID(ctx context.Context, id string) (*inventory.StockMovement, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_stock_movements WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: movement find: %w", err)
	}
	var m inventory.StockMovement
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, fmt.Errorf("inventory: movement decode: %w", err)
	}
	return &m, nil
}

func (r *sqliteRepository) UpdateMovement(ctx context.Context, m *inventory.StockMovement) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("inventory: movement marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE inventory_stock_movements SET data = ? WHERE id = ?`,
		string(data), m.ID)
	if err != nil {
		return fmt.Errorf("inventory: movement update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListMovements(ctx context.Context, itemCode, warehouseCode string, movementType inventory.MovementType, limit, offset int) ([]*inventory.StockMovement, int, error) {
	where := "1=1"
	args := []interface{}{}
	if itemCode != "" {
		where += " AND json_extract(data, '$.item_code') = ?"
		args = append(args, itemCode)
	}
	if warehouseCode != "" {
		where += " AND json_extract(data, '$.warehouse_code') = ?"
		args = append(args, warehouseCode)
	}
	if movementType != "" {
		where += " AND json_extract(data, '$.movement_type') = ?"
		args = append(args, movementType)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_stock_movements WHERE `+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: movement count: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM inventory_stock_movements WHERE `+where+` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: movement list: %w", err)
	}
	defer rows.Close()

	var movements []*inventory.StockMovement
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, 0, fmt.Errorf("inventory: movement scan: %w", err)
		}
		var m inventory.StockMovement
		if err := json.Unmarshal([]byte(data), &m); err != nil {
			return nil, 0, fmt.Errorf("inventory: movement decode: %w", err)
		}
		movements = append(movements, &m)
	}
	return movements, total, nil
}

func (r *sqliteRepository) NextMovementCode(ctx context.Context, prefix string) (int64, error) {
	return r.nextSeq(ctx, prefix+"_seq")
}

// --- Valuation Layer operations (FIFO) ---

func (r *sqliteRepository) CreateLayer(ctx context.Context, l *inventory.StockValuationLayer) error {
	data, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("inventory: layer marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO inventory_stock_valuation_layers (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		l.ID, string(data))
	if err != nil {
		return fmt.Errorf("inventory: layer create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) UpdateLayer(ctx context.Context, l *inventory.StockValuationLayer) error {
	data, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("inventory: layer marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE inventory_stock_valuation_layers SET data = ? WHERE id = ?`,
		string(data), l.ID)
	if err != nil {
		return fmt.Errorf("inventory: layer update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListLayersByItem(ctx context.Context, itemCode, warehouseCode string) ([]*inventory.StockValuationLayer, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM inventory_stock_valuation_layers
		 WHERE json_extract(data, '$.item_code') = ? AND json_extract(data, '$.warehouse_code') = ?
		 ORDER BY json_extract(data, '$.received_date') ASC, json_extract(data, '$.id') ASC`,
		itemCode, warehouseCode)
	if err != nil {
		return nil, fmt.Errorf("inventory: layer list: %w", err)
	}
	defer rows.Close()

	var layers []*inventory.StockValuationLayer
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("inventory: layer scan: %w", err)
		}
		var l inventory.StockValuationLayer
		if err := json.Unmarshal([]byte(data), &l); err != nil {
			return nil, fmt.Errorf("inventory: layer decode: %w", err)
		}
		layers = append(layers, &l)
	}
	return layers, rows.Err()
}

// --- Physical Count operations ---

func (r *sqliteRepository) CreatePhysicalCount(ctx context.Context, pc *inventory.PhysicalCount) error {
	data, err := json.Marshal(pc)
	if err != nil {
		return fmt.Errorf("inventory: physical count marshal: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO inventory_physical_counts (id, data) VALUES (?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		pc.ID, string(data))
	if err != nil {
		return fmt.Errorf("inventory: physical count create: %w", err)
	}
	return nil
}

func (r *sqliteRepository) FindPhysicalCountByID(ctx context.Context, id string) (*inventory.PhysicalCount, error) {
	var data string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM inventory_physical_counts WHERE id = ?`, id).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, inventory.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("inventory: physical count find: %w", err)
	}
	var pc inventory.PhysicalCount
	if err := json.Unmarshal([]byte(data), &pc); err != nil {
		return nil, fmt.Errorf("inventory: physical count decode: %w", err)
	}
	return &pc, nil
}

func (r *sqliteRepository) UpdatePhysicalCount(ctx context.Context, pc *inventory.PhysicalCount) error {
	data, err := json.Marshal(pc)
	if err != nil {
		return fmt.Errorf("inventory: physical count marshal: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE inventory_physical_counts SET data = ? WHERE id = ?`,
		string(data), pc.ID)
	if err != nil {
		return fmt.Errorf("inventory: physical count update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return inventory.ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListPhysicalCounts(ctx context.Context, warehouseCode string, status inventory.PhysicalCountStatus, limit, offset int) ([]*inventory.PhysicalCount, int, error) {
	where := "1=1"
	args := []interface{}{}
	if warehouseCode != "" {
		where += " AND json_extract(data, '$.warehouse_code') = ?"
		args = append(args, warehouseCode)
	}
	if status != "" {
		where += " AND json_extract(data, '$.status') = ?"
		args = append(args, status)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_physical_counts WHERE `+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: physical count count: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM inventory_physical_counts WHERE `+where+` ORDER BY json_extract(data, '$.created_at') DESC LIMIT ? OFFSET ?`,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("inventory: physical count list: %w", err)
	}
	defer rows.Close()

	var counts []*inventory.PhysicalCount
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, 0, fmt.Errorf("inventory: physical count scan: %w", err)
		}
		var pc inventory.PhysicalCount
		if err := json.Unmarshal([]byte(data), &pc); err != nil {
			return nil, 0, fmt.Errorf("inventory: physical count decode: %w", err)
		}
		counts = append(counts, &pc)
	}
	return counts, total, nil
}

func (r *sqliteRepository) NextPhysicalCountCode(ctx context.Context) (int64, error) {
	return r.nextSeq(ctx, "physical_count_seq")
}
