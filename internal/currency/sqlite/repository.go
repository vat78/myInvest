package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"myinvest/internal/currency"
)

// Repository borrows the connection pool; the application owns its lifetime.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

var _ currency.Repository = (*Repository)(nil)

func (s *Repository) ListCurrencies(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT code FROM currencies ORDER BY code")
	if err != nil {
		return nil, fmt.Errorf("загрузить валюты: %w", err)
	}
	defer rows.Close()
	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (s *Repository) AddCurrency(ctx context.Context, code string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO currencies(code) VALUES (?) ON CONFLICT(code) DO NOTHING", code)
	if err != nil {
		return fmt.Errorf("добавить валюту: %w", err)
	}
	return nil
}

func (s *Repository) FindRate(ctx context.Context, pair currency.Pair) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM currency_rates WHERE date = ? AND base = ? AND quote = ?", pair.Date, pair.Base, pair.Quote).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("загрузить курс: %w", err)
	}
	return value, true, nil
}

func (s *Repository) SaveRates(ctx context.Context, direct, inverse currency.Rate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("начать сохранение курсов: %w", err)
	}
	defer tx.Rollback()
	for _, rate := range []currency.Rate{direct, inverse} {
		_, err := tx.ExecContext(ctx, `INSERT INTO currency_rates(date, base, quote, value) VALUES (?, ?, ?, ?)
			ON CONFLICT(date, base, quote) DO UPDATE SET value = excluded.value`, rate.Date, rate.Base, rate.Quote, rate.Value)
		if err != nil {
			return fmt.Errorf("не удалось сохранить обе пары; проверьте, что обе валюты добавлены в справочник: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("завершить сохранение обеих пар: %w", err)
	}
	return nil
}
