package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"myinvest/internal/cashflow"
)

type querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
type Repository struct {
	db *sql.DB
	q  querier
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db, q: db} }
func (r *Repository) Transaction(ctx context.Context, fn func(cashflow.Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fn(&Repository{q: tx}); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) PortfolioCurrency(ctx context.Context, id string) (string, error) {
	var code string
	err := r.q.QueryRowContext(ctx, "SELECT currency_code FROM portfolios WHERE id=?", id).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		err = cashflow.ErrPortfolio
	}
	return code, err
}
func (r *Repository) CurrencyExists(ctx context.Context, code string) (bool, error) {
	var ok bool
	err := r.q.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code=?)", code).Scan(&ok)
	return ok, err
}

const columns = "id,portfolio_id,kind,date,currency_code,amount,portfolio_amount,comment"

func scan(row interface{ Scan(...any) error }) (cashflow.Operation, error) {
	var o cashflow.Operation
	err := row.Scan(&o.ID, &o.PortfolioID, &o.Kind, &o.Date, &o.CurrencyCode, &o.Amount, &o.PortfolioAmount, &o.Comment)
	if errors.Is(err, sql.ErrNoRows) {
		err = cashflow.ErrNotFound
	}
	return o, err
}
func (r *Repository) List(ctx context.Context, pid string) ([]cashflow.Operation, error) {
	rows, err := r.q.QueryContext(ctx, "SELECT "+columns+" FROM cash_flows WHERE portfolio_id=? ORDER BY date DESC,id DESC", pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []cashflow.Operation{}
	for rows.Next() {
		o, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (r *Repository) Find(ctx context.Context, pid, id string) (cashflow.Operation, error) {
	return scan(r.q.QueryRowContext(ctx, "SELECT "+columns+" FROM cash_flows WHERE portfolio_id=? AND id=?", pid, id))
}
func (r *Repository) Save(ctx context.Context, o cashflow.Operation) (cashflow.Operation, error) {
	if o.ID == "" {
		return scan(r.q.QueryRowContext(ctx, "INSERT INTO cash_flows(portfolio_id,kind,date,currency_code,amount,portfolio_amount,comment) VALUES (?,?,?,?,?,?,?) RETURNING "+columns, o.PortfolioID, o.Kind, o.Date, o.CurrencyCode, o.Amount, o.PortfolioAmount, o.Comment))
	}
	return scan(r.q.QueryRowContext(ctx, "UPDATE cash_flows SET kind=?,date=?,currency_code=?,amount=?,portfolio_amount=?,comment=? WHERE portfolio_id=? AND id=? RETURNING "+columns, o.Kind, o.Date, o.CurrencyCode, o.Amount, o.PortfolioAmount, o.Comment, o.PortfolioID, o.ID))
}
func (r *Repository) Delete(ctx context.Context, pid, id string) error {
	res, err := r.q.ExecContext(ctx, "DELETE FROM cash_flows WHERE portfolio_id=? AND id=?", pid, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return cashflow.ErrNotFound
	}
	return err
}
