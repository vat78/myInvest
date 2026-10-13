package desktop

import (
	"context"
	"errors"
	"fmt"
	"myinvest/internal/operations"
	"strconv"
)

type CashFlowInputDTO struct {
	Kind            string `json:"kind"`
	Date            string `json:"date"`
	CurrencyCode    string `json:"currencyCode"`
	Amount          string `json:"amount"`
	PortfolioAmount string `json:"portfolioAmount"`
	Comment         string `json:"comment"`
}
type CashFlowDTO struct {
	ID          string `json:"id"`
	PortfolioID string `json:"portfolioID"`
	CashFlowInputDTO
	Rate         string `json:"rate"`
	AssetID      string `json:"assetID"`
	Ticker       string `json:"ticker"`
	Quantity     string `json:"quantity"`
	Expenses     string `json:"expenses"`
	AveragePrice string `json:"averagePrice"`
	Rounded      bool   `json:"rounded"`
	MoneyChange  string `json:"moneyChange"`
}
type CashSnapshotDTO struct {
	CurrencyCode string           `json:"currencyCode"`
	Balance      string           `json:"balance"`
	Negative     bool             `json:"negative"`
	Operations   []CashFlowDTO    `json:"operations"`
	Warnings     []WarningDTO     `json:"warnings"`
	CashWarnings []CashWarningDTO `json:"cashWarnings"`
}
type CashResultDTO struct {
	Errors  map[string]string `json:"errors"`
	Message string            `json:"message"`
}
type CashRateDTO struct {
	Rate   string            `json:"rate"`
	Errors map[string]string `json:"errors"`
}
type CashFlowAPI struct {
	useService func(func(*operations.Service) error) error
}

func NewAPI(fn func(func(*operations.Service) error) error) *CashFlowAPI { return &CashFlowAPI{fn} }
func validID(id string) bool {
	v, err := strconv.ParseInt(id, 10, 64)
	return err == nil && v > 0 && strconv.FormatInt(v, 10) == id
}
func (a *CashFlowAPI) ListCashFlows(pid string) (CashSnapshotDTO, error) {
	out := CashSnapshotDTO{Operations: []CashFlowDTO{}, Warnings: []WarningDTO{}, CashWarnings: []CashWarningDTO{}}
	if !validID(pid) {
		return out, operations.ErrPortfolio
	}
	err := a.useService(func(s *operations.Service) error {
		snapshot, err := s.List(context.Background(), pid)
		if err != nil {
			return err
		}
		out.CurrencyCode = snapshot.CurrencyCode
		out.Balance = snapshot.Balance
		for _, w := range snapshot.CashWarnings {
			out.CashWarnings = append(out.CashWarnings, CashWarningDTO{w.Date, w.Balance})
		}
		for _, w := range snapshot.Warnings {
			out.Warnings = append(out.Warnings, WarningDTO{w.OperationID, w.Date, w.Ticker, w.Available, w.Quantity})
		}
		out.Negative = len(out.Balance) > 0 && out.Balance[0] == '-'
		for _, o := range snapshot.Operations {
			rate := ""
			var err error
			if !operations.IsTrade(o.Kind) {
				rate, err = operations.Rate(o.Amount, o.PortfolioAmount)
			}
			if err != nil {
				return err
			}
			out.Operations = append(out.Operations, CashFlowDTO{ID: o.ID, PortfolioID: o.PortfolioID, CashFlowInputDTO: CashFlowInputDTO{o.Kind, o.Date, o.CurrencyCode, o.Amount, o.PortfolioAmount, o.Comment}, Rate: rate, AssetID: o.AssetID, Ticker: o.Ticker, Quantity: o.Quantity, Expenses: o.Expenses, AveragePrice: o.AveragePrice, Rounded: o.Rounded, MoneyChange: operations.MoneyChange(o)})
		}
		return nil
	})
	if err != nil {
		return CashSnapshotDTO{}, fmt.Errorf("Не удалось загрузить операции. Повторите попытку")
	}
	return out, nil
}
func result(err error) CashResultDTO {
	out := CashResultDTO{Errors: map[string]string{}}
	var field *operations.FieldError
	if errors.As(err, &field) {
		out.Errors[field.Field] = field.Message
	} else if errors.Is(err, operations.ErrPortfolio) || errors.Is(err, operations.ErrNotFound) {
		out.Message = err.Error()
	} else if err != nil {
		out.Message = "Не удалось изменить операцию. Повторите попытку"
	}
	return out
}
func (a *CashFlowAPI) SaveCashFlow(pid, id string, in CashFlowInputDTO) CashResultDTO {
	if !validID(pid) {
		return result(operations.ErrPortfolio)
	}
	if id != "" && !validID(id) {
		return result(operations.ErrNotFound)
	}
	return result(a.useService(func(s *operations.Service) error {
		_, err := s.Save(context.Background(), pid, id, operations.Input{Kind: in.Kind, Date: in.Date, CurrencyCode: in.CurrencyCode, Amount: in.Amount, PortfolioAmount: in.PortfolioAmount, Comment: in.Comment})
		return err
	}))
}
func (a *CashFlowAPI) DeleteCashFlow(pid, id string) CashResultDTO {
	if !validID(pid) {
		return result(operations.ErrPortfolio)
	}
	if !validID(id) {
		return result(operations.ErrNotFound)
	}
	return result(a.useService(func(s *operations.Service) error { return s.Delete(context.Background(), pid, id) }))
}
func (a *CashFlowAPI) PreviewCashRate(amount, portfolioAmount string) CashRateDTO {
	out := CashRateDTO{Errors: map[string]string{}}
	if _, err := operations.Amount(amount); err != nil {
		out.Errors["amount"] = err.Error()
	}
	if _, err := operations.Amount(portfolioAmount); err != nil {
		out.Errors["portfolioAmount"] = err.Error()
	}
	if len(out.Errors) == 0 {
		out.Rate, _ = operations.Rate(amount, portfolioAmount)
	}
	return out
}

type WarningDTO struct {
	OperationID string `json:"operationID"`
	Date        string `json:"date"`
	Ticker      string `json:"ticker"`
	Available   string `json:"available"`
	Quantity    string `json:"quantity"`
}
type TradeInputDTO struct {
	Kind     string `json:"kind"`
	Date     string `json:"date"`
	AssetID  string `json:"assetID"`
	Quantity string `json:"quantity"`
	Amount   string `json:"amount"`
	Expenses string `json:"expenses"`
	Comment  string `json:"comment"`
}
type TradePreviewDTO struct {
	AveragePrice string            `json:"averagePrice"`
	Rounded      bool              `json:"rounded"`
	MoneyChange  string            `json:"moneyChange"`
	Errors       map[string]string `json:"errors"`
}
type AssetDTO struct {
	ID     string `json:"id"`
	Ticker string `json:"ticker"`
}
type AssetResultDTO struct {
	Asset   AssetDTO          `json:"asset"`
	Errors  map[string]string `json:"errors"`
	Message string            `json:"message"`
}

func (a *CashFlowAPI) ListAssets() ([]AssetDTO, error) {
	out := []AssetDTO{}
	err := a.useService(func(s *operations.Service) error {
		assets, err := s.ListAssets(context.Background())
		if err != nil {
			return err
		}
		for _, v := range assets {
			out = append(out, AssetDTO{v.ID, v.Ticker})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Не удалось загрузить активы. Повторите попытку")
	}
	return out, nil
}
func (a *CashFlowAPI) AddAsset(ticker string) AssetResultDTO {
	var asset operations.Asset
	err := a.useService(func(s *operations.Service) error {
		var err error
		asset, err = s.AddAsset(context.Background(), ticker)
		return err
	})
	r := result(err)
	if err != nil && len(r.Errors) == 0 {
		r.Message = "Не удалось добавить актив. Повторите попытку"
	}
	return AssetResultDTO{AssetDTO{asset.ID, asset.Ticker}, r.Errors, r.Message}
}
func (a *CashFlowAPI) SaveTrade(pid, id string, in TradeInputDTO) CashResultDTO {
	if !validID(pid) {
		return result(operations.ErrPortfolio)
	}
	if id != "" && !validID(id) {
		return result(operations.ErrNotFound)
	}
	if !validID(in.AssetID) {
		return result(&operations.FieldError{Field: "assetID", Message: "Выберите актив"})
	}
	if !operations.IsTrade(in.Kind) {
		return result(&operations.FieldError{Field: "kind", Message: "Выберите покупку или продажу"})
	}
	return result(a.useService(func(s *operations.Service) error {
		_, err := s.Save(context.Background(), pid, id, operations.Input{Kind: in.Kind, Date: in.Date, AssetID: in.AssetID, Quantity: in.Quantity, Amount: in.Amount, Expenses: in.Expenses, Comment: in.Comment})
		return err
	}))
}
func (a *CashFlowAPI) PreviewTrade(kind, quantity, amount, expenses string) TradePreviewDTO {
	out := TradePreviewDTO{Errors: map[string]string{}}
	in, err := operations.ValidateTrade(operations.Input{Kind: kind, Date: "2000-01-01", AssetID: "1", Quantity: quantity, Amount: amount, Expenses: expenses})
	if err != nil {
		r := result(err)
		out.Errors = r.Errors
		return out
	}
	out.AveragePrice, out.Rounded, _ = operations.AveragePrice(in.Amount, in.Quantity)
	out.MoneyChange = operations.MoneyChange(operations.Operation{Input: in})
	return out
}

type CashWarningDTO struct {
	Date    string `json:"date"`
	Balance string `json:"balance"`
}
