package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"myinvest/internal/planning"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db} }

var _ planning.Repository = (*Repository)(nil)

func ferr(f, m string) error { return &planning.FieldError{Field: f, Message: m} }
func directory(ctx context.Context, tx *sql.Tx, code, bid string) (string, []planning.IndexPoint, error) {
	var currency string
	e := tx.QueryRowContext(ctx, "SELECT code FROM currencies WHERE code=?", code).Scan(&currency)
	if errors.Is(e, sql.ErrNoRows) {
		return "", nil, ferr("currencyCode", "Валюта не найдена")
	}
	if e != nil {
		return "", nil, e
	}
	var exists bool
	e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM benchmarks WHERE id=?)", bid).Scan(&exists)
	if errors.Is(e, sql.ErrNoRows) || (e == nil && !exists) {
		return "", nil, ferr("benchmarkID", "Выберите бенчмарк из справочника")
	}
	if e != nil {
		return "", nil, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT date,value FROM benchmark_points WHERE benchmark_id=? ORDER BY date", bid)
	if e != nil {
		return "", nil, e
	}
	defer rows.Close()
	points := []planning.IndexPoint{}
	for rows.Next() {
		var p planning.IndexPoint
		if e = rows.Scan(&p.Date, &p.Value); e != nil {
			return "", nil, e
		}
		points = append(points, p)
	}
	return currency, points, rows.Err()
}
func read(ctx context.Context, tx *sql.Tx, code string) (planning.Plan, error) {
	out := planning.Plan{}
	p := &out.Parameters
	p.CurrencyCode = code
	var num, den, original string
	e := tx.QueryRowContext(ctx, `SELECT CAST(revision AS TEXT),CAST(start_year AS TEXT),CAST(years AS TEXT),mode,goal,contribution,return_percent,inflation_percent,CAST(benchmark_id AS TEXT),allow_estimated,initial_capital,initial_numerator,initial_denominator,original FROM currency_financial_plans WHERE currency_code=?`, code).Scan(&p.Revision, &p.StartYear, &p.Years, &p.Mode, &p.Goal, &p.Contribution, &p.ReturnPercent, &p.InflationPercent, &p.BenchmarkID, &p.AllowEstimated, &p.InitialCapital, &num, &den, &original)
	if errors.Is(e, sql.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	out.Initial, _ = new(big.Rat).SetString(num + "/" + den)
	if out.Initial == nil {
		return out, fmt.Errorf("invalid stored capital")
	}
	e = json.Unmarshal([]byte(original), &out.Original)
	out.Found = e == nil
	return out, e
}
func (r *Repository) Load(ctx context.Context, code string) (planning.Plan, []planning.IndexPoint, error) {
	tx, e := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return planning.Plan{}, nil, e
	}
	defer tx.Rollback()
	var exists bool
	if e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code=?)", code).Scan(&exists); e != nil {
		return planning.Plan{}, nil, e
	}
	if !exists {
		return planning.Plan{}, nil, ferr("currencyCode", "Валюта не найдена")
	}
	out, e := read(ctx, tx, code)
	if e != nil {
		return out, nil, e
	}
	if !out.Found {
		return out, nil, tx.Commit()
	}
	currency, points, e := directory(ctx, tx, code, out.Parameters.BenchmarkID)
	if e != nil {
		return out, nil, e
	}
	out.Currency = currency
	return out, points, tx.Commit()
}
func (r *Repository) Process(ctx context.Context, p planning.Parameters, save bool, calculate func(planning.Parameters, string, *big.Rat, []planning.IndexPoint) (planning.Scenario, error)) (planning.Plan, error) {
	out := planning.Plan{}
	tx, e := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: !save})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if save { // Acquire the writer before reading any inputs or revision.
		if _, e = tx.ExecContext(ctx, "UPDATE currencies SET code=code WHERE code=?", p.CurrencyCode); e != nil {
			return out, e
		}
	}
	previous, e := read(ctx, tx, p.CurrencyCode)
	if e != nil {
		return out, e
	}
	if save && p.Revision != previous.Parameters.Revision {
		return out, ferr("revision", "План изменён. Загрузите сохранённый план и повторите изменение")
	}
	currency, points, e := directory(ctx, tx, p.CurrencyCode, p.BenchmarkID)
	if e != nil {
		return out, e
	}
	initial, _ := new(big.Rat).SetString(p.InitialCapital)
	current, e := calculate(p, currency, initial, points)
	if e != nil {
		return out, e
	}
	out = planning.Plan{Found: save, Parameters: p, Currency: currency, Initial: initial, Current: current}
	if !save {
		return out, tx.Commit()
	}
	original, e := json.Marshal(current)
	if e != nil {
		return out, e
	}
	e = tx.QueryRowContext(ctx, `INSERT INTO currency_financial_plans(currency_code,revision,start_year,years,mode,goal,contribution,return_percent,inflation_percent,benchmark_id,allow_estimated,initial_capital,initial_numerator,initial_denominator,original)
 VALUES(?,1,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(currency_code) DO UPDATE SET revision=currency_financial_plans.revision+1,start_year=excluded.start_year,years=excluded.years,mode=excluded.mode,goal=excluded.goal,contribution=excluded.contribution,return_percent=excluded.return_percent,inflation_percent=excluded.inflation_percent,benchmark_id=excluded.benchmark_id,allow_estimated=excluded.allow_estimated,initial_capital=excluded.initial_capital,initial_numerator=excluded.initial_numerator,initial_denominator=excluded.initial_denominator,original=excluded.original RETURNING CAST(revision AS TEXT)`, p.CurrencyCode, p.StartYear, p.Years, p.Mode, p.Goal, p.Contribution, p.ReturnPercent, p.InflationPercent, p.BenchmarkID, p.AllowEstimated, p.InitialCapital, initial.Num().String(), initial.Denom().String(), string(original)).Scan(&out.Parameters.Revision)
	if e != nil {
		return out, e
	}
	out.Original = current
	return out, tx.Commit()
}
