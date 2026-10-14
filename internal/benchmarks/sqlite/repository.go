package sqlite

import (
	"context"
	"database/sql"
	"myinvest/internal/benchmarks"
	"strings"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db} }

var _ benchmarks.Repository = (*Repository)(nil)

func (r *Repository) List(ctx context.Context) ([]benchmarks.Benchmark, error) {
	rows, e := r.db.QueryContext(ctx, `SELECT CAST(id AS TEXT),name,COALESCE(currency_code,'') FROM benchmarks ORDER BY name_key,id`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []benchmarks.Benchmark{}
	for rows.Next() {
		var b benchmarks.Benchmark
		if e = rows.Scan(&b.ID, &b.Name, &b.CurrencyCode); e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (r *Repository) Create(ctx context.Context, b benchmarks.Benchmark) (benchmarks.Benchmark, error) {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return b, e
	}
	defer tx.Rollback()
	if b.CurrencyCode != "" {
		var exists bool
		if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM currencies WHERE code=?)`, b.CurrencyCode).Scan(&exists); e != nil {
			return b, e
		}
		if !exists {
			return b, &benchmarks.FieldError{Field: "currencyCode", Message: "Выберите валюту из справочника"}
		}
	}
	key := strings.ToLower(b.Name)
	var exists bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM benchmarks WHERE name_key=?)`, key).Scan(&exists); e != nil {
		return b, e
	}
	if exists {
		return b, &benchmarks.FieldError{Field: "name", Message: "Бенчмарк с таким названием уже существует"}
	}
	e = tx.QueryRowContext(ctx, `INSERT INTO benchmarks(name,name_key,currency_code) VALUES(?,?,NULLIF(?,'')) RETURNING CAST(id AS TEXT)`, b.Name, key, b.CurrencyCode).Scan(&b.ID)
	if e != nil {
		return b, e
	}
	return b, tx.Commit()
}

type reader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func history(ctx context.Context, db reader, id string) ([]benchmarks.Point, error) {
	rows, e := db.QueryContext(ctx, `SELECT CAST(benchmark_id AS TEXT),date,mode,input,value,COALESCE(growth,'') FROM benchmark_points WHERE benchmark_id=? ORDER BY date`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []benchmarks.Point{}
	for rows.Next() {
		var p benchmarks.Point
		if e = rows.Scan(&p.BenchmarkID, &p.Date, &p.Mode, &p.Input, &p.Value, &p.Growth); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *Repository) History(ctx context.Context, id string) ([]benchmarks.Point, error) {
	return history(ctx, r.db, id)
}
func (r *Repository) Rewrite(ctx context.Context, id string, calculate func([]benchmarks.Point) ([]benchmarks.Point, error)) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Acquire the write lock before reading the history, preventing lost updates.
	res, e := tx.ExecContext(ctx, `UPDATE benchmarks SET name=name WHERE id=?`, id)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return &benchmarks.FieldError{Field: "benchmarkID", Message: "Выберите бенчмарк из справочника"}
	}
	points, e := history(ctx, tx, id)
	if e != nil {
		return e
	}
	points, e = calculate(points)
	if e != nil {
		return e
	}
	for _, p := range points {
		_, e = tx.ExecContext(ctx, `INSERT INTO benchmark_points(benchmark_id,date,mode,input,value,growth) VALUES(?,?,?,?,?,NULLIF(?,'')) ON CONFLICT(benchmark_id,date) DO UPDATE SET mode=excluded.mode,input=excluded.input,value=excluded.value,growth=excluded.growth`, id, p.Date, p.Mode, p.Input, p.Value, p.Growth)
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}
