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
	Rate string `json:"rate"`
}
type CashSnapshotDTO struct {
	CurrencyCode string        `json:"currencyCode"`
	Balance      string        `json:"balance"`
	Negative     bool          `json:"negative"`
	Operations   []CashFlowDTO `json:"operations"`
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
	out := CashSnapshotDTO{Operations: []CashFlowDTO{}}
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
		out.Negative = len(out.Balance) > 0 && out.Balance[0] == '-'
		for _, o := range snapshot.Operations {
			rate, err := cashflow.Rate(o.Amount, o.PortfolioAmount)
			if err != nil {
				return err
			}
			out.Operations = append(out.Operations, CashFlowDTO{o.ID, o.PortfolioID, CashFlowInputDTO{o.Kind, o.Date, o.CurrencyCode, o.Amount, o.PortfolioAmount, o.Comment}, rate})
		}
		return nil
	})
	if err != nil {
		return CashSnapshotDTO{}, fmt.Errorf("Не удалось загрузить денежные операции. Повторите попытку")
	}
	return out, nil
}
func result(err error) CashResultDTO {
	out := CashResultDTO{Errors: map[string]string{}}
	var field *cashflow.FieldError
	if errors.As(err, &field) {
		out.Errors[field.Field] = field.Message
	} else if errors.Is(err, cashflow.ErrPortfolio) || errors.Is(err, cashflow.ErrNotFound) {
		out.Message = err.Error()
	} else if err != nil {
		out.Message = "Не удалось изменить денежную операцию. Повторите попытку"
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
