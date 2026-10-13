package desktop

import (
	"context"
	"errors"
	"fmt"
	"myinvest/internal/cashflow"
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
	Rate           string `json:"rate"`
	AssetID        string `json:"assetID"`
	Ticker         string `json:"ticker"`
	Quantity       string `json:"quantity"`
	Expenses       string `json:"expenses"`
	AveragePrice   string `json:"averagePrice"`
	Rounded        bool   `json:"rounded"`
	MoneyChange    string `json:"moneyChange"`
	OldUnits       string `json:"oldUnits"`
	NewUnits       string `json:"newUnits"`
	QuantityBefore string `json:"quantityBefore"`
	QuantityAfter  string `json:"quantityAfter"`
	ReverseSplit   bool   `json:"reverseSplit"`
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
	useService func(func(*cashflow.Service) error) error
}

func NewAPI(fn func(func(*cashflow.Service) error) error) *CashFlowAPI { return &CashFlowAPI{fn} }
func validID(id string) bool {
	v, err := strconv.ParseInt(id, 10, 64)
	return err == nil && v > 0 && strconv.FormatInt(v, 10) == id
}
func (a *CashFlowAPI) ListCashFlows(pid string) (CashSnapshotDTO, error) {
	out := CashSnapshotDTO{Operations: []CashFlowDTO{}, Warnings: []WarningDTO{}, CashWarnings: []CashWarningDTO{}}
	if !validID(pid) {
		return out, cashflow.ErrPortfolio
	}
	err := a.useService(func(s *cashflow.Service) error {
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
			if !cashflow.IsTrade(o.Kind) && o.Kind != "split" {
				rate, err = cashflow.Rate(o.Amount, o.PortfolioAmount)
			}
			if err != nil {
				return err
			}
			out.Operations = append(out.Operations, CashFlowDTO{ID: o.ID, PortfolioID: o.PortfolioID, CashFlowInputDTO: CashFlowInputDTO{o.Kind, o.Date, o.CurrencyCode, o.Amount, o.PortfolioAmount, o.Comment}, Rate: rate, AssetID: o.AssetID, Ticker: o.Ticker, Quantity: o.Quantity, Expenses: o.Expenses, AveragePrice: o.AveragePrice, Rounded: o.Rounded, MoneyChange: cashflow.MoneyChange(o), OldUnits: o.OldUnits, NewUnits: o.NewUnits, QuantityBefore: o.QuantityBefore, QuantityAfter: o.QuantityAfter, ReverseSplit: cashflow.IsReverseSplit(o)})
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
	var split *cashflow.SplitError
	var field *cashflow.FieldError
	if errors.As(err, &field) {
		out.Errors[field.Field] = field.Message
	} else if errors.As(err, &split) {
		out.Message = split.Error()
	} else if errors.Is(err, cashflow.ErrPortfolio) || errors.Is(err, cashflow.ErrNotFound) {
		out.Message = err.Error()
	} else if err != nil {
		out.Message = "Не удалось изменить операцию. Повторите попытку"
	}
	return out
}
func (a *CashFlowAPI) SaveCashFlow(pid, id string, in CashFlowInputDTO) CashResultDTO {
	if !validID(pid) {
		return result(cashflow.ErrPortfolio)
	}
	if id != "" && !validID(id) {
		return result(cashflow.ErrNotFound)
	}
	return result(a.useService(func(s *cashflow.Service) error {
		_, err := s.Save(context.Background(), pid, id, cashflow.Input{Kind: in.Kind, Date: in.Date, CurrencyCode: in.CurrencyCode, Amount: in.Amount, PortfolioAmount: in.PortfolioAmount, Comment: in.Comment})
		return err
	}))
}
func (a *CashFlowAPI) DeleteCashFlow(pid, id string) CashResultDTO {
	if !validID(pid) {
		return result(cashflow.ErrPortfolio)
	}
	if !validID(id) {
		return result(cashflow.ErrNotFound)
	}
	return result(a.useService(func(s *cashflow.Service) error { return s.Delete(context.Background(), pid, id) }))
}
func (a *CashFlowAPI) PreviewCashRate(amount, portfolioAmount string) CashRateDTO {
	out := CashRateDTO{Errors: map[string]string{}}
	if _, err := cashflow.Amount(amount); err != nil {
		out.Errors["amount"] = err.Error()
	}
	if _, err := cashflow.Amount(portfolioAmount); err != nil {
		out.Errors["portfolioAmount"] = err.Error()
	}
	if len(out.Errors) == 0 {
		out.Rate, _ = cashflow.Rate(amount, portfolioAmount)
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
	err := a.useService(func(s *cashflow.Service) error {
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
	var asset cashflow.Asset
	err := a.useService(func(s *cashflow.Service) error {
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
		return result(cashflow.ErrPortfolio)
	}
	if id != "" && !validID(id) {
		return result(cashflow.ErrNotFound)
	}
	if !validID(in.AssetID) {
		return result(&cashflow.FieldError{Field: "assetID", Message: "Выберите актив"})
	}
	if !cashflow.IsTrade(in.Kind) {
		return result(&cashflow.FieldError{Field: "kind", Message: "Выберите покупку или продажу"})
	}
	return result(a.useService(func(s *cashflow.Service) error {
		_, err := s.Save(context.Background(), pid, id, cashflow.Input{Kind: in.Kind, Date: in.Date, AssetID: in.AssetID, Quantity: in.Quantity, Amount: in.Amount, Expenses: in.Expenses, Comment: in.Comment})
		return err
	}))
}
func (a *CashFlowAPI) PreviewTrade(kind, quantity, amount, expenses string) TradePreviewDTO {
	out := TradePreviewDTO{Errors: map[string]string{}}
	in, err := cashflow.ValidateTrade(cashflow.Input{Kind: kind, Date: "2000-01-01", AssetID: "1", Quantity: quantity, Amount: amount, Expenses: expenses})
	if err != nil {
		r := result(err)
		out.Errors = r.Errors
		return out
	}
	out.AveragePrice, out.Rounded, _ = cashflow.AveragePrice(in.Amount, in.Quantity)
	out.MoneyChange = cashflow.MoneyChange(cashflow.Operation{Input: in})
	return out
}

type CashWarningDTO struct {
	Date    string `json:"date"`
	Balance string `json:"balance"`
}

type SplitInputDTO struct {
	Date     string `json:"date"`
	AssetID  string `json:"assetID"`
	OldUnits string `json:"oldUnits"`
	NewUnits string `json:"newUnits"`
	Comment  string `json:"comment"`
}
type SplitPreviewDTO struct {
	QuantityBefore string            `json:"quantityBefore"`
	QuantityAfter  string            `json:"quantityAfter"`
	Errors         map[string]string `json:"errors"`
	Message        string            `json:"message"`
}

func splitInput(in SplitInputDTO) cashflow.Input {
	return cashflow.Input{Kind: "split", Date: in.Date, AssetID: in.AssetID, OldUnits: in.OldUnits, NewUnits: in.NewUnits, Comment: in.Comment}
}
func splitIDs(pid, id string, in SplitInputDTO) error {
	if !validID(pid) {
		return cashflow.ErrPortfolio
	}
	if id != "" && !validID(id) {
		return cashflow.ErrNotFound
	}
	if !validID(in.AssetID) {
		return &cashflow.FieldError{Field: "assetID", Message: "Выберите актив"}
	}
	return nil
}
func (a *CashFlowAPI) SaveSplit(pid, id string, in SplitInputDTO) CashResultDTO {
	if err := splitIDs(pid, id, in); err != nil {
		return result(err)
	}
	return result(a.useService(func(s *cashflow.Service) error {
		_, err := s.Save(context.Background(), pid, id, splitInput(in))
		return err
	}))
}
func (a *CashFlowAPI) PreviewSplit(pid, id string, in SplitInputDTO) SplitPreviewDTO {
	var q cashflow.SplitQuantities
	err := splitIDs(pid, id, in)
	if err == nil {
		err = a.useService(func(s *cashflow.Service) error {
			var err error
			q, err = s.PreviewSplit(context.Background(), pid, id, splitInput(in))
			return err
		})
	}
	r := result(err)
	return SplitPreviewDTO{q.Before, q.After, r.Errors, r.Message}
}
