package sqlite

import (
	"context"
	"database/sql"
	"myinvest/internal/quotes"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db} }

var _ quotes.Repository = (*Repository)(nil)

func (r *Repository) List(ctx context.Context) ([]quotes.Quote, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT CAST(q.asset_id AS TEXT),q.date,q.currency_code,q.price,a.ticker
 FROM asset_quotes q JOIN assets a ON a.id=q.asset_id ORDER BY q.date DESC,a.ticker,q.currency_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []quotes.Quote{}
	for rows.Next() {
		var q quotes.Quote
		if err := rows.Scan(&q.AssetID, &q.Date, &q.CurrencyCode, &q.Price, &q.Ticker); err != nil {
			return nil, err
		}
		result = append(result, q)
	}
	return result, rows.Err()
}
func (r *Repository) Save(ctx context.Context, q quotes.Quote) error {
	// Validate referenced codes and write under one transaction; foreign keys are the final guard.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id=?)", q.AssetID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return &quotes.FieldError{Field: "assetID", Message: "Выберите актив из справочника"}
	}
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code=?)", q.CurrencyCode).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return &quotes.FieldError{Field: "currencyCode", Message: "Выберите валюту из справочника"}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO asset_quotes(asset_id,date,currency_code,price) VALUES(?,?,?,?)
 ON CONFLICT(asset_id,date,currency_code) DO UPDATE SET price=excluded.price`, q.AssetID, q.Date, q.CurrencyCode, q.Price)
	if err != nil {
		return err
	}
	return tx.Commit()
}
