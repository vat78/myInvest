// Package desktop adapts portfolio use cases to the Wails bridge.
package desktop

import (
	"context"
	"errors"
	"fmt"
	"myinvest/internal/portfolio"
	"strconv"
)

type PortfolioDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OpenedOn     string `json:"openedOn"`
	CurrencyCode string `json:"currencyCode"`
}
type PortfolioInputDTO struct {
	Name         string `json:"name"`
	OpenedOn     string `json:"openedOn"`
	CurrencyCode string `json:"currencyCode"`
}
type SavePortfolioResultDTO struct {
	Portfolio PortfolioDTO      `json:"portfolio"`
	Errors    map[string]string `json:"errors"`
	Message   string            `json:"message"`
}

// A distinct type name avoids collisions with the currency API in Wails bindings.
type PortfolioAPI struct {
	useService func(func(*portfolio.Service) error) error
}

func NewAPI(useService func(func(*portfolio.Service) error) error) *PortfolioAPI {
	return &PortfolioAPI{useService}
}
func dto(p portfolio.Portfolio) PortfolioDTO {
	return PortfolioDTO{p.ID, p.Name, p.OpenedOn, p.CurrencyCode}
}
func (a *PortfolioAPI) ListPortfolios() ([]PortfolioDTO, error) {
	result := []PortfolioDTO{}
	err := a.useService(func(s *portfolio.Service) error {
		rows, err := s.List(context.Background())
		if err != nil {
			return err
		}
		for _, p := range rows {
			result = append(result, dto(p))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Не удалось загрузить портфели. Повторите попытку")
	}
	return result, nil
}
func (a *PortfolioAPI) save(id string, input PortfolioInputDTO) SavePortfolioResultDTO {
	result := SavePortfolioResultDTO{Errors: map[string]string{}}
	err := a.useService(func(s *portfolio.Service) error {
		data := portfolio.Input{Name: input.Name, OpenedOn: input.OpenedOn, CurrencyCode: input.CurrencyCode}
		var p portfolio.Portfolio
		var err error
		if id == "" {
			p, err = s.Create(context.Background(), data)
		} else {
			p, err = s.Update(context.Background(), id, data)
		}
		if err == nil {
			result.Portfolio = dto(p)
		}
		return err
	})
	var field *portfolio.FieldError
	if errors.As(err, &field) {
		result.Errors[field.Field] = field.Message
	} else if errors.Is(err, portfolio.ErrNotFound) {
		result.Message = portfolio.ErrNotFound.Error()
	} else if err != nil {
		result.Message = "Не удалось сохранить портфель. Повторите попытку"
	}
	return result
}
func (a *PortfolioAPI) CreatePortfolio(input PortfolioInputDTO) SavePortfolioResultDTO {
	return a.save("", input)
}
func (a *PortfolioAPI) UpdatePortfolio(id string, input PortfolioInputDTO) SavePortfolioResultDTO {
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil || value <= 0 || strconv.FormatInt(value, 10) != id {
		return SavePortfolioResultDTO{Errors: map[string]string{}, Message: portfolio.ErrNotFound.Error()}
	}
	return a.save(id, input)
}
