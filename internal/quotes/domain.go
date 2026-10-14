// Package quotes defines manually entered prices of one asset unit.
package quotes

import (
	"myinvest/internal/currency"
	"strconv"
	"time"
)

type Quote struct {
	AssetID      string `json:"assetID"`
	Date         string `json:"date"`
	CurrencyCode string `json:"currencyCode"`
	Price        string `json:"price"`
	Ticker       string `json:"ticker"`
}
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Message }

func Validate(q Quote) (Quote, error) {
	id, err := strconv.ParseInt(q.AssetID, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != q.AssetID {
		return q, &FieldError{"assetID", "Выберите актив из справочника"}
	}
	d, err := time.Parse("2006-01-02", q.Date)
	if err != nil || d.Year() < 1 || d.Format("2006-01-02") != q.Date {
		return q, &FieldError{"date", "Введите существующую календарную дату YYYY-MM-DD"}
	}
	q.CurrencyCode, err = currency.Code(q.CurrencyCode)
	if err != nil {
		return q, &FieldError{"currencyCode", "Выберите валюту из справочника"}
	}
	q.Price, err = currency.Decimal(q.Price)
	if err != nil {
		return q, &FieldError{"price", "Введите цену больше нуля: цифры и точка или запятая"}
	}
	q.Ticker = "" // Display code always comes from the directory.
	return q, nil
}
