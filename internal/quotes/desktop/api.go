package desktop

import (
	"context"
	"errors"
	"fmt"
	"myinvest/internal/quotes"
)

type QuoteDTO struct {
	AssetID      string `json:"assetID"`
	Date         string `json:"date"`
	CurrencyCode string `json:"currencyCode"`
	Price        string `json:"price"`
	Ticker       string `json:"ticker"`
}
type QuoteResultDTO struct {
	Errors  map[string]string `json:"errors"`
	Message string            `json:"message"`
}
type QuotesAPI struct {
	useService func(func(*quotes.Service) error) error
}

func NewAPI(fn func(func(*quotes.Service) error) error) *QuotesAPI { return &QuotesAPI{fn} }
func (a *QuotesAPI) ListQuotes() ([]QuoteDTO, error) {
	out := []QuoteDTO{}
	err := a.useService(func(s *quotes.Service) error {
		rows, err := s.List(context.Background())
		if err != nil {
			return err
		}
		for _, q := range rows {
			out = append(out, QuoteDTO{q.AssetID, q.Date, q.CurrencyCode, q.Price, q.Ticker})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Не удалось загрузить котировки. Повторите попытку")
	}
	return out, nil
}
func (a *QuotesAPI) SaveQuote(in QuoteDTO) QuoteResultDTO {
	out := QuoteResultDTO{Errors: map[string]string{}}
	err := a.useService(func(s *quotes.Service) error {
		return s.Save(context.Background(), quotes.Quote{AssetID: in.AssetID, Date: in.Date, CurrencyCode: in.CurrencyCode, Price: in.Price})
	})
	var field *quotes.FieldError
	if errors.As(err, &field) {
		out.Errors[field.Field] = field.Message
	} else if err != nil {
		out.Message = "Не удалось сохранить котировку. Повторите попытку"
	}
	return out
}
