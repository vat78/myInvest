package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"myinvest/internal/operations"
	operationssqlite "myinvest/internal/operations/sqlite"
	"myinvest/internal/valuation"
)

func (r *Repository) LoadAllocation(ctx context.Context, pid, date string) (valuation.AllocationSnapshot, error) {
	out := valuation.AllocationSnapshot{Plan: valuation.AllocationPlan{Targets: []valuation.Target{}}, Assets: []operations.Asset{}}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out.Analysis.Currency, out.Analysis.Operations, err = operationssqlite.ReadHistoryInTransaction(ctx, tx, pid, date)
	if err != nil {
		return out, err
	}
	out.Analysis.Pricing, err = loadPricing(ctx, tx, date, false)
	if err != nil {
		return out, err
	}
	var cash int64
	err = tx.QueryRowContext(ctx, `SELECT generation,CAST(revision AS TEXT),cash_weight_units FROM portfolio_allocation_plans WHERE portfolio_id=?`, pid).Scan(&out.Plan.Generation, &out.Plan.Revision, &cash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	out.Plan.CashPercent = valuation.PercentString(cash)
	rows, err := tx.QueryContext(ctx, `SELECT CAST(a.id AS TEXT),a.ticker,coalesce(a.type_code,''),t.weight_units FROM assets a LEFT JOIN portfolio_allocation_targets t ON t.asset_id=a.id AND t.portfolio_id=? ORDER BY a.ticker,a.id`, pid)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var a operations.Asset
		var units sql.NullInt64
		if err = rows.Scan(&a.ID, &a.Ticker, &a.TypeCode, &units); err != nil {
			rows.Close()
			return out, err
		}
		out.Assets = append(out.Assets, a)
		if units.Valid {
			out.Plan.Targets = append(out.Plan.Targets, valuation.Target{AssetID: a.ID, Percent: valuation.PercentString(units.Int64)})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}
func (r *Repository) SaveAllocation(ctx context.Context, pid string, p valuation.AllocationPlan) error {
	p, err := valuation.ValidateAllocation(p)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	cash, _ := valuation.PercentUnits(p.CashPercent)
	var result sql.Result
	if p.Generation == "" {
		result, err = tx.ExecContext(ctx, `INSERT INTO portfolio_allocation_plans(portfolio_id,generation,revision,cash_weight_units) VALUES(?,?,1,?) ON CONFLICT(portfolio_id) DO NOTHING`, pid, rand.Text(), cash)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE portfolio_allocation_plans SET revision=revision+1,cash_weight_units=? WHERE portfolio_id=? AND generation=? AND revision=?`, cash, pid, p.Generation, p.Revision)
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return valuation.ErrAllocationConflict
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM portfolio_allocation_targets WHERE portfolio_id=?`, pid); err != nil {
		return err
	}
	for i, t := range p.Targets {
		var exists int
		if err = tx.QueryRowContext(ctx, `SELECT 1 FROM assets WHERE id=?`, t.AssetID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return &valuation.AllocationFieldError{Field: fmt.Sprintf("targets.%d.assetID", i), Message: "Актив не найден"}
		} else if err != nil {
			return err
		}
		units, _ := valuation.PercentUnits(t.Percent)
		if _, err = tx.ExecContext(ctx, `INSERT INTO portfolio_allocation_targets(portfolio_id,asset_id,weight_units) VALUES(?,?,?)`, pid, t.AssetID, units); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (r *Repository) DeleteAllocation(ctx context.Context, pid, generation, revision string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM portfolio_allocation_plans WHERE portfolio_id=? AND generation=? AND revision=?`, pid, generation, revision)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return valuation.ErrAllocationConflict
	}
	return nil
}
