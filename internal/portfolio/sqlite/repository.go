package sqlite

import (
	"context"
	"database/sql"
	"errors"
	sqliteDriver "modernc.org/sqlite"
	"myinvest/internal/portfolio"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db} }

var _ portfolio.Repository = (*Repository)(nil)

const columns = "id, name, name_key, opened_on, currency_code"

func scan(row interface{ Scan(...any) error }) (portfolio.Portfolio, error) {
	var p portfolio.Portfolio
	err := row.Scan(&p.ID, &p.Name, &p.NameKey, &p.OpenedOn, &p.CurrencyCode)
	if errors.Is(err, sql.ErrNoRows) {
		return p, portfolio.ErrNotFound
	}
	return p, err
}
func (r *Repository) List(ctx context.Context) ([]portfolio.Portfolio, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+columns+" FROM portfolios ORDER BY opened_on DESC, name_key ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []portfolio.Portfolio{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func (r *Repository) Find(ctx context.Context, id string) (portfolio.Portfolio, error) {
	return scan(r.db.QueryRowContext(ctx, "SELECT "+columns+" FROM portfolios WHERE id = ?", id))
}
func (r *Repository) CurrencyExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code = ?)", code).Scan(&exists)
	return exists, err
}
func (r *Repository) NameExists(ctx context.Context, key, exceptID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM portfolios WHERE name_key = ? AND id != ?)", key, exceptID).Scan(&exists)
	return exists, err
}
func writeError(err error) error {
	var dbErr *sqliteDriver.Error
	if errors.As(err, &dbErr) {
		switch dbErr.Code() {
		case 2067:
			return portfolio.DuplicateName() // SQLITE_CONSTRAINT_UNIQUE
		case 787:
			return &portfolio.FieldError{Field: "currencyCode", Message: "Выберите валюту из справочника"}
		}
	}
	return err
}
func (r *Repository) Create(ctx context.Context, p portfolio.Portfolio) (portfolio.Portfolio, error) {
	result, err := scan(r.db.QueryRowContext(ctx, "INSERT INTO portfolios(name, name_key, opened_on, currency_code) VALUES (?, ?, ?, ?) RETURNING "+columns, p.Name, p.NameKey, p.OpenedOn, p.CurrencyCode))
	return result, writeError(err)
}
func (r *Repository) Update(ctx context.Context, p portfolio.Portfolio) (portfolio.Portfolio, error) {
	// A single statement is atomic; currency is neither assigned nor allowed to differ.
	result, err := scan(r.db.QueryRowContext(ctx, "UPDATE portfolios SET name = ?, name_key = ?, opened_on = ? WHERE id = ? AND currency_code = ? RETURNING "+columns, p.Name, p.NameKey, p.OpenedOn, p.ID, p.CurrencyCode))
	return result, writeError(err)
}
