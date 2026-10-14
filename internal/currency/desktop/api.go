// Package desktop adapts currency use cases to the Wails bridge.
package desktop

import (
	"context"
	"fmt"

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
	DisplayValue string `json:"displayValue"`
	Value        string `json:"value"`
	Found        bool   `json:"found"`
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
		if err == nil && result.Found {
			result.DisplayValue, err = currency.DisplayRate(result.Value)
		}
		return err
	})
	return
}
func (a *API) SaveCurrencyRate(input SaveCurrencyRateDTO) error {
	return a.useService(func(s *currency.Service) error {
		return s.Save(context.Background(), input.Date, input.Base, input.Quote, input.Value)
	})
}

// ListCurrencyRates returns exact stored values, including inverse pairs.
func (a *API) ListCurrencyRates() ([]SaveCurrencyRateDTO, error) {
	out := []SaveCurrencyRateDTO{}
	err := a.useService(func(s *currency.Service) error {
		rows, err := s.ListRates(context.Background())
		if err != nil {
			return err
		}
		for _, r := range rows {
			out = append(out, SaveCurrencyRateDTO{r.Date, r.Base, r.Quote, r.Value})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Не удалось загрузить курсы валют. Повторите попытку")
	}
	return out, nil
}
