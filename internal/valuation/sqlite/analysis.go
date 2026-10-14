package sqlite

import (
	"context"
	"database/sql"
	operationssqlite "myinvest/internal/operations/sqlite"
	"myinvest/internal/valuation"
)

func (r *Repository) LoadAnalysis(ctx context.Context, pid, date string) (valuation.AnalysisSnapshot, error) {
	out := valuation.AnalysisSnapshot{}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out.Currency, out.Operations, err = operationssqlite.ReadHistoryInTransaction(ctx, tx, pid, date)
	if err != nil {
		return out, err
	}
	out.Pricing, err = loadPricing(ctx, tx, date, false)
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}
