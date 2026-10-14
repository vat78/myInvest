package sqlite

import (
	"context"
	"database/sql"
	"myinvest/internal/quotes"
	"myinvest/internal/valuation"
)

func loadPricing(ctx context.Context, tx *sql.Tx, date string, includeSplits bool) (valuation.Pricing, error) {
	out := valuation.Pricing{}
	rows, err := tx.QueryContext(ctx, "SELECT CAST(asset_id AS TEXT),date,currency_code,price FROM asset_quotes WHERE date<=? ORDER BY asset_id,date,currency_code", date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var q quotes.Quote
		if err = rows.Scan(&q.AssetID, &q.Date, &q.CurrencyCode, &q.Price); err != nil {
			rows.Close()
			return out, err
		}
		out.Quotes = append(out.Quotes, q)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT date,base,quote,value FROM currency_rates WHERE date<=? ORDER BY date", date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var f valuation.ExchangeRate
		if err = rows.Scan(&f.Date, &f.Base, &f.Quote, &f.Value); err != nil {
			rows.Close()
			return out, err
		}
		out.FX = append(out.FX, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if !includeSplits {
		return out, nil // Analysis already has its split history.
	}
	// Only split metadata is needed. Persisted quantities and trade values remain authoritative.
	query := `SELECT CAST(o.portfolio_id AS TEXT),CAST(s.asset_id AS TEXT),o.date,s.old_units,s.new_units
 FROM split_details s JOIN operations o ON o.id=s.operation_id WHERE o.date<=?`
	rows, err = tx.QueryContext(ctx, query+" ORDER BY o.date,o.id", date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var s valuation.PriceSplit
		if err = rows.Scan(&s.PortfolioID, &s.AssetID, &s.Date, &s.OldUnits, &s.NewUnits); err != nil {
			rows.Close()
			return out, err
		}
		out.Splits = append(out.Splits, s)
	}
	err = rows.Err()
	rows.Close()
	return out, err
}
