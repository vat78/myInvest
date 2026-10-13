// Package desktop adapts currency use cases to the Wails bridge.
package desktop

import (
	"context"

	"myinvest/internal/currency"
)

type CurrencyPairDTO struct {
	Date  string `json:"date"`
	Base  string `json:"base"`
	Quote string `json:"quote"`
}
type SaveCurrencyRateDTO struct {
	Date  string `json:"date"`
	Base  string `json:"base"`
	Quote string `json:"quote"`
	Value string `json:"value"`
}
type CurrencyRateDTO struct {
	Value string `json:"value"`
	Found bool   `json:"found"`
}

// The application provides access guarded by its storage lifecycle.
type API struct {
	useService func(func(*currency.Service) error) error
}

func NewAPI(useService func(func(*currency.Service) error) error) *API {
	return &API{useService: useService}
}

func (a *API) ListCurrencies() (codes []string, err error) {
	err = a.useService(func(s *currency.Service) error {
		codes, err = s.List(context.Background())
		return err
	})
	return
}
func (a *API) AddCurrency(code string) (normalized string, err error) {
	err = a.useService(func(s *currency.Service) error {
		normalized, err = s.Add(context.Background(), code)
		return err
	})
	return
}
func (a *API) GetCurrencyRate(input CurrencyPairDTO) (result CurrencyRateDTO, err error) {
	err = a.useService(func(s *currency.Service) error {
		result.Value, result.Found, err = s.Find(context.Background(), input.Date, input.Base, input.Quote)
		return err
	})
	return
}
func (a *API) SaveCurrencyRate(input SaveCurrencyRateDTO) error {
	return a.useService(func(s *currency.Service) error {
		return s.Save(context.Background(), input.Date, input.Base, input.Quote, input.Value)
	})
}
