package desktop

import (
	"context"
	"errors"
	"myinvest/internal/valuation"
)

type OverviewAPI struct {
	withService func(func(*valuation.Service) error) error
}

func NewAPI(withService func(func(*valuation.Service) error) error) *OverviewAPI {
	return &OverviewAPI{withService}
}
func (a *OverviewAPI) GetOverview(date string) (valuation.Overview, error) {
	if err := valuation.ValidateDate(date); err != nil {
		return valuation.Overview{}, err
	}
	var out valuation.Overview
	err := a.withService(func(s *valuation.Service) error {
		var err error
		out, err = s.Overview(context.Background(), date)
		return err
	})
	if err != nil {
		return valuation.Overview{}, errors.New("Не удалось загрузить обзор. Повторите попытку")
	}
	return out, nil
}

func (a *OverviewAPI) GetPortfolioAnalysis(pid, date string) (valuation.Analysis, error) {
	if err := valuation.ValidatePortfolioID(pid); err != nil {
		return valuation.Analysis{}, err
	}
	if err := valuation.ValidateDate(date); err != nil {
		return valuation.Analysis{}, err
	}
	var out valuation.Analysis
	err := a.withService(func(s *valuation.Service) error {
		var err error
		out, err = s.Analysis(context.Background(), pid, date)
		return err
	})
	if err != nil {
		return valuation.Analysis{}, errors.New("Не удалось загрузить анализ портфеля. Повторите попытку")
	}
	return out, nil
}
