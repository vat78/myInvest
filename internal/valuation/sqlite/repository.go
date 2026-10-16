package sqlite

import (
	"context"
	"database/sql"
	"myinvest/internal/valuation"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db} }
func (r *Repository) Load(ctx context.Context, date string) (valuation.Snapshot, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return valuation.Snapshot{}, err
	}
	defer tx.Rollback()
	out, err := loadSnapshot(ctx, tx, date)
	if err != nil {
		return out, err
	}
	return out, tx.Commit()
}

func loadSnapshot(ctx context.Context, tx *sql.Tx, date string) (valuation.Snapshot, error) {
	out := valuation.Snapshot{Portfolios: []valuation.Portfolio{}, Rates: []valuation.Rate{}}
	rows, err := tx.QueryContext(ctx, "SELECT id,name,currency_code FROM portfolios ORDER BY name_key,id")
	if err != nil {
		return out, err
	}
	indexes := map[string]int{}
	for rows.Next() {
		var p valuation.Portfolio
		if err = rows.Scan(&p.ID, &p.Name, &p.Currency); err != nil {
			rows.Close()
			return out, err
		}
		p.Days = []valuation.Day{}
		indexes[p.ID] = len(out.Portfolios)
		out.Portfolios = append(out.Portfolios, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT c.portfolio_id,c.date,c.balance,c.deposits_rub,a.ticker,coalesce(a.type_code,''),b.quantity,b.value_numerator,b.value_denominator,CAST(b.asset_id AS TEXT),source.date
 FROM daily_cash_balances c LEFT JOIN daily_asset_balances b ON b.portfolio_id=c.portfolio_id AND b.date=c.date
 LEFT JOIN assets a ON a.id=b.asset_id LEFT JOIN operations source ON source.id=b.price_operation_id WHERE c.date<=? ORDER BY c.portfolio_id,c.date,b.asset_id`, date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var pid, day, cash string
		var ticker, code, q, num, den, deposits, assetID, tradeDate sql.NullString
		if err = rows.Scan(&pid, &day, &cash, &deposits, &ticker, &code, &q, &num, &den, &assetID, &tradeDate); err != nil {
			rows.Close()
			return out, err
		}
		p := &out.Portfolios[indexes[pid]]
		if len(p.Days) == 0 || p.Days[len(p.Days)-1].Date != day {
			p.Days = append(p.Days, valuation.Day{Date: day, Cash: cash, Deposits: deposits.String, Positions: []valuation.Position{}})
		}
		if ticker.Valid {
			d := &p.Days[len(p.Days)-1]
			d.Positions = append(d.Positions, valuation.Position{AssetID: assetID.String, TradeDate: tradeDate.String, Ticker: ticker.String, TypeCode: code.String, Quantity: q.String, Numerator: num.String, Denominator: den.String})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT date,base,value FROM currency_rates WHERE quote='RUB' AND date<=? ORDER BY date", date)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var rate valuation.Rate
		if err = rows.Scan(&rate.Date, &rate.Currency, &rate.Value); err != nil {
			rows.Close()
			return out, err
		}
		out.Rates = append(out.Rates, rate)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	out.Pricing, err = loadPricing(ctx, tx, date, true)
	if err != nil {
		return out, err
	}
	return out, nil
}
