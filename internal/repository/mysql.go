package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"assessment.local/returns/internal/domain"
	"github.com/go-sql-driver/mysql"
)

type MySQL struct{ db *sql.DB }

func OpenMySQL(ctx context.Context, dsn string) (*MySQL, error) {
	if dsn == "" {
		return nil, errors.New("DATABASE_DSN is required")
	}
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("invalid DATABASE_DSN")
	}
	config.ParseTime = true
	config.Loc = time.FixedZone("Europe/Moscow", 3*60*60)
	config.Timeout, config.ReadTimeout, config.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	if config.Params == nil {
		config.Params = map[string]string{}
	}
	config.Params["time_zone"] = "'+03:00'"
	connector, err := mysql.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("open MySQL: %w", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect MySQL: %w", err)
	}
	return &MySQL{db: db}, nil
}

func (m *MySQL) Close() error                   { return m.db.Close() }
func (m *MySQL) Ping(ctx context.Context) error { return m.db.PingContext(ctx) }

func sqlTimestamp(column string) string {
	return "DATE_FORMAT(" + column + ", '%Y-%m-%dT%H:%i:%s+03:00')"
}
func sqlDate(column string) string { return "DATE_FORMAT(" + column + ", '%Y-%m-%d')" }

func readRows[T any](ctx context.Context, tx *sql.Tx, query string, scan func(*sql.Rows) (T, error)) ([]T, error) {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]T, 0)
	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (m *MySQL) Read(ctx context.Context) (domain.Dataset, error) {
	var data domain.Dataset
	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return data, err
	}
	defer tx.Rollback()
	var hashes []byte
	err = tx.QueryRowContext(ctx, "SELECT version, "+sqlTimestamp("as_of")+", timezone, currency, "+sqlDate("default_from")+", "+sqlDate("default_to")+", source_hashes FROM dataset_metadata WHERE id = 1").Scan(&data.Manifest.Version, &data.Manifest.AsOf, &data.Manifest.Timezone, &data.Manifest.Currency, &data.Manifest.From, &data.Manifest.To, &hashes)
	if err != nil {
		return data, fmt.Errorf("read dataset metadata: %w", err)
	}
	if err := json.Unmarshal(hashes, &data.Manifest.Files); err != nil {
		return data, err
	}
	data.Suppliers, err = readRows(ctx, tx, "SELECT id,name FROM suppliers ORDER BY id", func(rows *sql.Rows) (domain.Supplier, error) {
		var item domain.Supplier
		err := rows.Scan(&item.ID, &item.Name)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read suppliers: %w", err)
	}
	data.Policies, err = readRows(ctx, tx, "SELECT id,supplier_id,"+sqlDate("valid_from")+","+sqlDate("valid_to")+",confirmed,window_days,starts_at,ends_at,include_first_day,calendar,requires_packaging,accepts_good_quality FROM supplier_policies ORDER BY id", func(rows *sql.Rows) (domain.Policy, error) {
		var item domain.Policy
		err := rows.Scan(&item.ID, &item.SupplierID, &item.ValidFrom, &item.ValidTo, &item.Confirmed, &item.WindowDays, &item.StartsAt, &item.EndsAt, &item.IncludeFirstDay, &item.Calendar, &item.Packaging, &item.AcceptsGood)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read supplier policies: %w", err)
	}
	data.Returns, err = readRows(ctx, tx, "SELECT id,return_id,supplier_id,policy_id,customer_id,product,quantity,unit_book_cost_minor,"+sqlTimestamp("supplier_shipped_at")+","+sqlTimestamp("requested_at")+","+sqlTimestamp("warehouse_received_at")+",packaging_complete,quality,posted FROM return_lines ORDER BY id", func(rows *sql.Rows) (domain.ReturnLine, error) {
		var item domain.ReturnLine
		err := rows.Scan(&item.ID, &item.ReturnID, &item.SupplierID, &item.PolicyID, &item.CustomerID, &item.Product, &item.Quantity, &item.UnitBookCostMinor, &item.SupplierShippedAt, &item.RequestedAt, &item.WarehouseReceivedAt, &item.PackagingComplete, &item.Quality, &item.Posted)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read return lines: %w", err)
	}
	data.SupplierReturns, err = readRows(ctx, tx, "SELECT id,supplier_id,"+sqlTimestamp("dispatched_at")+",posted FROM supplier_returns ORDER BY id", func(rows *sql.Rows) (domain.SupplierReturn, error) {
		item := domain.SupplierReturn{Lines: []domain.Allocation{}}
		err := rows.Scan(&item.ID, &item.SupplierID, &item.DispatchedAt, &item.Posted)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read supplier returns: %w", err)
	}
	type linkedAllocation struct {
		ParentID string
		Line     domain.Allocation
	}
	allocations, err := readRows(ctx, tx, "SELECT supplier_return_id,id,return_line_id,quantity FROM supplier_return_lines ORDER BY id", func(rows *sql.Rows) (linkedAllocation, error) {
		var item linkedAllocation
		err := rows.Scan(&item.ParentID, &item.Line.ID, &item.Line.ReturnLineID, &item.Line.Quantity)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read supplier return lines: %w", err)
	}
	parents := make(map[string]int, len(data.SupplierReturns))
	for index, item := range data.SupplierReturns {
		parents[item.ID] = index
	}
	for _, item := range allocations {
		index, found := parents[item.ParentID]
		if !found {
			return data, fmt.Errorf("unknown supplier return %s", item.ParentID)
		}
		data.SupplierReturns[index].Lines = append(data.SupplierReturns[index].Lines, item.Line)
	}
	data.Events, err = readRows(ctx, tx, "SELECT event_id,allocation_id,type,"+sqlTimestamp("occurred_at")+","+sqlTimestamp("recorded_at")+",quantity,reason,reverses_event_id FROM return_events ORDER BY delivery_id", func(rows *sql.Rows) (domain.Event, error) {
		var item domain.Event
		err := rows.Scan(&item.ID, &item.AllocationID, &item.Type, &item.OccurredAt, &item.RecordedAt, &item.Quantity, &item.Reason, &item.ReversesEventID)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read return events: %w", err)
	}
	data.StockMovements, err = readRows(ctx, tx, "SELECT id,return_line_id,quantity_delta,value_delta_minor,"+sqlTimestamp("occurred_at")+","+sqlTimestamp("recorded_at")+" FROM stock_movements ORDER BY id", func(rows *sql.Rows) (domain.StockMovement, error) {
		var item domain.StockMovement
		err := rows.Scan(&item.ID, &item.ReturnLineID, &item.QuantityDelta, &item.ValueDeltaMinor, &item.OccurredAt, &item.RecordedAt)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read stock movements: %w", err)
	}
	data.Financial, err = readRows(ctx, tx, "SELECT id,return_line_id,type,amount_minor,"+sqlTimestamp("occurred_at")+" FROM financial_entries ORDER BY id", func(rows *sql.Rows) (domain.FinancialEntry, error) {
		var item domain.FinancialEntry
		err := rows.Scan(&item.ID, &item.ReturnLineID, &item.Type, &item.AmountMinor, &item.OccurredAt)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read financial entries: %w", err)
	}
	data.Shipments, err = readRows(ctx, tx, "SELECT id,supplier_id,"+sqlTimestamp("shipped_at")+",quantity,amount_minor FROM shipments ORDER BY id", func(rows *sql.Rows) (domain.Shipment, error) {
		var item domain.Shipment
		err := rows.Scan(&item.ID, &item.SupplierID, &item.ShippedAt, &item.Quantity, &item.AmountMinor)
		return item, err
	})
	if err != nil {
		return data, fmt.Errorf("read shipments: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return data, err
	}
	return data, nil
}
