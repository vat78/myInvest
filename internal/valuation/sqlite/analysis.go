package sqlite

import (
	"context"
	operationssqlite "myinvest/internal/operations/sqlite"
	"myinvest/internal/valuation"
)

func (r *Repository) LoadAnalysis(ctx context.Context, pid, date string) (valuation.AnalysisSnapshot, error) {
	code, ops, err := operationssqlite.NewRepository(r.db).ReadHistory(ctx, pid, date)
	return valuation.AnalysisSnapshot{Currency: code, Operations: ops}, err
}
