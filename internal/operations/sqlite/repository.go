package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"myinvest/internal/operations"
)

type querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
type Repository struct {
	db           *sql.DB
	q            querier
	legacySchema bool
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db, q: db} }
func (r *Repository) Transaction(ctx context.Context, fn func(operations.Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Acquire the SQLite writer reservation before reading history, preventing
	// lost updates and deferred read-to-write upgrade failures. No rows change.
	if _, err = tx.ExecContext(ctx, "UPDATE portfolios SET id=id WHERE 0"); err != nil {
		return err
	}
	if err = fn(&Repository{q: tx}); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) PortfolioCurrency(ctx context.Context, id string) (string, error) {
	var code string
	err := r.q.QueryRowContext(ctx, "SELECT currency_code FROM portfolios WHERE id=?", id).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		err = operations.ErrPortfolio
	}
	return code, err
}
func (r *Repository) CurrencyExists(ctx context.Context, code string) (bool, error) {
	var ok bool
	err := r.q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code=?)", code).Scan(&ok)
	return ok, err
}

const legacyColumns = `o.id,o.portfolio_id,o.kind,o.date,o.comment,
 coalesce(c.currency_code,''),coalesce(c.amount,t.amount,''),coalesce(c.portfolio_amount,''),
 coalesce(t.asset_id,''),coalesce(t.quantity,''),coalesce(t.expenses,''),coalesce(t.average_price,''),coalesce(a.ticker,''),
 c.operation_id,t.operation_id`
const legacyJoins = ` FROM operations o LEFT JOIN cash_flow_details c ON c.operation_id=o.id
 LEFT JOIN trade_details t ON t.operation_id=o.id LEFT JOIN assets a ON a.id=t.asset_id `

const columns = `o.id,o.portfolio_id,o.kind,o.date,o.comment,
 coalesce(c.currency_code,''),coalesce(c.amount,t.amount,''),coalesce(c.portfolio_amount,''),
 coalesce(t.asset_id,s.asset_id,''),coalesce(t.quantity,''),coalesce(t.expenses,''),coalesce(t.average_price,''),coalesce(a.ticker,''),
 c.operation_id,t.operation_id,coalesce(s.old_units,''),coalesce(s.new_units,''),s.operation_id`
const joins = ` FROM operations o LEFT JOIN cash_flow_details c ON c.operation_id=o.id
 LEFT JOIN trade_details t ON t.operation_id=o.id LEFT JOIN split_details s ON s.operation_id=o.id
 LEFT JOIN assets a ON a.id=coalesce(t.asset_id,s.asset_id) `

func (r *Repository) selection() string {
	if r.legacySchema {
		return legacyColumns + ",'','',NULL" + legacyJoins
	}
	return columns + joins
}

func scan(row interface{ Scan(...any) error }) (operations.Operation, error) {
	var o operations.Operation
	var cashID, tradeID, splitID sql.NullString
	err := row.Scan(&o.ID, &o.PortfolioID, &o.Kind, &o.Date, &o.Comment, &o.CurrencyCode, &o.Amount, &o.PortfolioAmount, &o.AssetID, &o.Quantity, &o.Expenses, &o.AveragePrice, &o.Ticker, &cashID, &tradeID, &o.OldUnits, &o.NewUnits, &splitID)
	if errors.Is(err, sql.ErrNoRows) {
		return o, operations.ErrNotFound
	}
	if err != nil {
		return o, err
	}
	if o.Kind == "split" {
		if !splitID.Valid || cashID.Valid || tradeID.Valid || o.Ticker == "" {
			return o, errors.New("incomplete split details")
		}
		in, err := operations.ValidateSplit(o.Input)
		if err != nil {
			return o, err
		}
		o.Input = in
	} else if operations.IsTrade(o.Kind) {
		if !tradeID.Valid || cashID.Valid || splitID.Valid || o.Ticker == "" {
			return o, errors.New("incomplete trade details")
		}
		in, err := operations.ValidateTrade(o.Input)
		if err != nil {
			return o, err
		}
		o.Input = in
		price, rounded, err := operations.AveragePrice(o.Amount, o.Quantity)
		if err != nil {
			return o, err
		}
		if price != o.AveragePrice {
			return o, errors.New("invalid stored average price")
		}
		o.Rounded = rounded
	} else {
		if !cashID.Valid || tradeID.Valid || splitID.Valid || (o.Kind != "deposit" && o.Kind != "withdrawal") {
			return o, errors.New("incomplete cash details")
		}
		if _, err := operations.Amount(o.Amount); err != nil {
			return o, err
		}
		if _, err := operations.Amount(o.PortfolioAmount); err != nil {
			return o, err
		}
	}
	return o, nil
}
func (r *Repository) List(ctx context.Context, pid string) ([]operations.Operation, error) {
	rows, err := r.q.QueryContext(ctx, "SELECT "+r.selection()+"WHERE o.portfolio_id=? ORDER BY o.date DESC,CASE WHEN o.kind='split' THEN 0 ELSE 1 END,o.id DESC", pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []operations.Operation{}
	for rows.Next() {
		o, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (r *Repository) Find(ctx context.Context, pid, id string) (operations.Operation, error) {
	return scan(r.q.QueryRowContext(ctx, "SELECT "+r.selection()+"WHERE o.portfolio_id=? AND o.id=?", pid, id))
}
func (r *Repository) Save(ctx context.Context, o operations.Operation) (operations.Operation, error) {
	var err error
	if o.ID == "" {
		err = r.q.QueryRowContext(ctx, "INSERT INTO operations(portfolio_id,kind,date,comment) VALUES (?,?,?,?) RETURNING id", o.PortfolioID, o.Kind, o.Date, o.Comment).Scan(&o.ID)
	} else {
		res, e := r.q.ExecContext(ctx, "UPDATE operations SET kind=?,date=?,comment=? WHERE portfolio_id=? AND id=?", o.Kind, o.Date, o.Comment, o.PortfolioID, o.ID)
		err = e
		if err == nil {
			n, e := res.RowsAffected()
			err = e
			if n == 0 {
				err = operations.ErrNotFound
			}
		}
	}
	if err != nil {
		return o, err
	}
	if o.Kind == "split" {
		_, err = r.q.ExecContext(ctx, `INSERT INTO split_details(operation_id,asset_id,old_units,new_units) VALUES (?,?,?,?)
 ON CONFLICT(operation_id) DO UPDATE SET asset_id=excluded.asset_id,old_units=excluded.old_units,new_units=excluded.new_units`, o.ID, o.AssetID, o.OldUnits, o.NewUnits)
	} else if operations.IsTrade(o.Kind) {
		_, err = r.q.ExecContext(ctx, `INSERT INTO trade_details(operation_id,asset_id,quantity,amount,expenses,average_price) VALUES (?,?,?,?,?,?)
  ON CONFLICT(operation_id) DO UPDATE SET asset_id=excluded.asset_id,quantity=excluded.quantity,amount=excluded.amount,expenses=excluded.expenses,average_price=excluded.average_price`, o.ID, o.AssetID, o.Quantity, o.Amount, o.Expenses, o.AveragePrice)
	} else {
		_, err = r.q.ExecContext(ctx, `INSERT INTO cash_flow_details(operation_id,currency_code,amount,portfolio_amount) VALUES (?,?,?,?)
  ON CONFLICT(operation_id) DO UPDATE SET currency_code=excluded.currency_code,amount=excluded.amount,portfolio_amount=excluded.portfolio_amount`, o.ID, o.CurrencyCode, o.Amount, o.PortfolioAmount)
	}
	if err != nil {
		return o, err
	}
	if err = r.rebuild(ctx, o.PortfolioID); err != nil {
		return o, err
	}
	return r.Find(ctx, o.PortfolioID, o.ID)
}
func (r *Repository) Delete(ctx context.Context, pid, id string) error {
	res, err := r.q.ExecContext(ctx, "DELETE FROM operations WHERE portfolio_id=? AND id=?", pid, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return operations.ErrNotFound
	}
	return r.rebuild(ctx, pid)
}
func (r *Repository) ListAssets(ctx context.Context) ([]operations.Asset, error) {
	rows, err := r.q.QueryContext(ctx, "SELECT id,ticker,coalesce(type_code,'') FROM assets ORDER BY ticker")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []operations.Asset{}
	for rows.Next() {
		var a operations.Asset
		if err := rows.Scan(&a.ID, &a.Ticker, &a.TypeCode); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Repository) AssetExists(ctx context.Context, id string) (bool, error) {
	var found bool
	err := r.q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id=?)", id).Scan(&found)
	return found, err
}
func (r *Repository) AddAsset(ctx context.Context, ticker, typeCode string) (operations.Asset, error) {
	var out operations.Asset
	var exists bool
	if err := r.q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE ticker=?)", ticker).Scan(&exists); err != nil {
		return out, err
	}
	if exists {
		return out, &operations.FieldError{Field: "ticker", Message: "Такой тикер уже существует"}
	}
	err := r.q.QueryRowContext(ctx, "INSERT INTO assets(ticker,type_code) VALUES (?,?) RETURNING id,ticker,type_code", ticker, typeCode).Scan(&out.ID, &out.Ticker, &out.TypeCode)
	return out, err
}
func (r *Repository) rebuild(ctx context.Context, pid string) error {
	ops, err := r.List(ctx, pid)
	if err != nil {
		return err
	}
	history, err := operations.Rebuild(ops)
	if err != nil {
		return err
	}
	if _, err = r.q.ExecContext(ctx, "DELETE FROM daily_cash_balances WHERE portfolio_id=?", pid); err != nil {
		return err
	}
	var valued bool
	if err = r.q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pragma_table_info('daily_asset_balances') WHERE name='value_numerator')").Scan(&valued); err != nil {
		return err
	}
	var depositsEnabled bool
	if err = r.q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pragma_table_info('daily_cash_balances') WHERE name='deposits_rub')").Scan(&depositsEnabled); err != nil {
		return err
	}
	var deposits map[string]string
	if depositsEnabled {
		currency, e := r.PortfolioCurrency(ctx, pid)
		if e != nil {
			return e
		}
		deposits, err = operations.DepositTotals(ops, currency)
		if err != nil {
			return err
		}
	}
	for _, day := range history.Days {
		if _, err = r.q.ExecContext(ctx, "INSERT INTO daily_cash_balances(portfolio_id,date,balance) VALUES (?,?,?)", pid, day.Date, day.Balance); err != nil {
			return err
		}
		if depositsEnabled {
			var value any
			if deposits[day.Date] != "" {
				value = deposits[day.Date]
			}
			if _, err = r.q.ExecContext(ctx, "UPDATE daily_cash_balances SET deposits_rub=? WHERE portfolio_id=? AND date=?", value, pid, day.Date); err != nil {
				return err
			}
		}
		for aid, q := range day.Quantities {
			if valued {
				value, available := day.Values[aid]
				var num, den, source any
				if available {
					num, den = value.Numerator, value.Denominator
					if value.PriceOperationID != "" {
						source = value.PriceOperationID
					}
				}
				_, err = r.q.ExecContext(ctx, "INSERT INTO daily_asset_balances(portfolio_id,date,asset_id,quantity,value_numerator,value_denominator,price_operation_id) VALUES (?,?,?,?,?,?,?)", pid, day.Date, aid, q, num, den, source)
			} else {
				_, err = r.q.ExecContext(ctx, "INSERT INTO daily_asset_balances(portfolio_id,date,asset_id,quantity) VALUES (?,?,?,?)", pid, day.Date, aid, q)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RebuildAll is the exact data step of migration 0005. It shares the live writer's
// calculation and persistence and runs in the same transaction as the SQL.
func RebuildAll(ctx context.Context, tx *sql.Tx) error {
	// Migration 0005 executes before split_details exists.
	var hasSplits bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name='split_details' AND type='table')").Scan(&hasSplits); err != nil {
		return err
	}
	r := &Repository{q: tx, legacySchema: !hasSplits}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM portfolios")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := r.rebuild(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) UpdateAssetType(ctx context.Context, id, typeCode string) (operations.Asset, error) {
	var out operations.Asset
	err := r.q.QueryRowContext(ctx, "UPDATE assets SET type_code=? WHERE id=? RETURNING id,ticker,type_code", typeCode, id).Scan(&out.ID, &out.Ticker, &out.TypeCode)
	if errors.Is(err, sql.ErrNoRows) {
		return out, operations.ErrAssetNotFound
	}
	return out, err
}
