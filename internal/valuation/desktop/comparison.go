package desktop

import (
	"context"
	"errors"
	"myinvest/internal/valuation"
)

type ComparisonAPI struct {
	withService func(func(*valuation.ComparisonService) error) error
}

func NewComparisonAPI(fn func(func(*valuation.ComparisonService) error) error) *ComparisonAPI {
	return &ComparisonAPI{fn}
}
func (a *ComparisonAPI) GetReturnComparison(inflationID, date string) (valuation.ReturnComparison, error) {
	if err := valuation.ValidateDate(date); err != nil {
		return valuation.ReturnComparison{}, err
	}
	if err := valuation.ValidatePortfolioID(inflationID); err != nil {
		return valuation.ReturnComparison{}, errors.New("Выберите бенчмарк инфляции")
	}
	var out valuation.ReturnComparison
	err := a.withService(func(s *valuation.ComparisonService) error {
		var e error
		out, e = s.Compare(context.Background(), inflationID, date)
		return e
	})
	if err != nil {
		return valuation.ReturnComparison{}, errors.New("Не удалось рассчитать сравнение. Проверьте историю выбранной инфляции и повторите попытку")
	}
	return out, nil
}
