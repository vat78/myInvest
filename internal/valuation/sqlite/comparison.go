package sqlite

import (
	"context"
	"database/sql"
	"myinvest/internal/benchmarks"
	"myinvest/internal/valuation"
)

func (r *Repository) LoadComparison(ctx context.Context, date string) (valuation.ComparisonSnapshot, error) {
	out := valuation.ComparisonSnapshot{Benchmarks: []valuation.ComparisonBenchmark{}, Flows: []valuation.ComparisonFlow{}}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out.Snapshot, err = loadSnapshot(ctx, tx, date)
	if err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT CAST(id AS TEXT),name,coalesce(currency_code,'') FROM benchmarks ORDER BY name_key,id`)
	if err != nil {
		return out, err
	}
	indexes := map[string]int{}
	for rows.Next() {
		var b valuation.ComparisonBenchmark
		if err = rows.Scan(&b.Benchmark.ID, &b.Benchmark.Name, &b.Benchmark.CurrencyCode); err != nil {
			rows.Close()
			return out, err
		}
		b.Points = []benchmarks.Point{}
		indexes[b.Benchmark.ID] = len(out.Benchmarks)
		out.Benchmarks = append(out.Benchmarks, b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT CAST(benchmark_id AS TEXT),date,value FROM benchmark_points WHERE date<=? ORDER BY benchmark_id,date`, date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var p benchmarks.Point
		if err = rows.Scan(&p.BenchmarkID, &p.Date, &p.Value); err != nil {
			rows.Close()
			return out, err
		}
		i := indexes[p.BenchmarkID]
		out.Benchmarks[i].Points = append(out.Benchmarks[i].Points, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT CAST(o.portfolio_id AS TEXT),o.date,o.kind,c.portfolio_amount FROM operations o JOIN cash_flow_details c ON c.operation_id=o.id WHERE o.date<=? ORDER BY o.portfolio_id,o.date,o.id`, date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var f valuation.ComparisonFlow
		if err = rows.Scan(&f.PortfolioID, &f.Date, &f.Kind, &f.Amount); err != nil {
			rows.Close()
			return out, err
		}
		out.Flows = append(out.Flows, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}
