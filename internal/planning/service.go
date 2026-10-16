package planning

import (
	"context"
	"math/big"
	"myinvest/internal/currency"
)

type Plan struct {
	Found      bool       `json:"found"`
	Parameters Parameters `json:"parameters"`
	Currency   string     `json:"currency"`
	Original   Scenario   `json:"original"`
	Current    Scenario   `json:"current"`
	Initial    *big.Rat   `json:"-"`
}
type Repository interface {
	Process(context.Context, Parameters, bool, func(Parameters, string, *big.Rat, []IndexPoint) (Scenario, error)) (Plan, error)
	Load(context.Context, string) (Plan, []IndexPoint, error)
}
type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{r} }
func (s *Service) process(ctx context.Context, p Parameters, today string, save bool) (Plan, error) {
	p, e := Validate(p)
	if e != nil {
		return Plan{}, e
	}
	return s.repo.Process(ctx, p, save, func(p Parameters, currency string, initial *big.Rat, points []IndexPoint) (Scenario, error) {
		factors, e := Resolve(p, points, today)
		if e != nil {
			return Scenario{}, e
		}
		return Calculate(p, initial, factors)
	})
}
func (s *Service) Preview(ctx context.Context, p Parameters, today string) (Plan, error) {
	return s.process(ctx, p, today, false)
}
func (s *Service) Save(ctx context.Context, p Parameters, today string) (Plan, error) {
	return s.process(ctx, p, today, true)
}
func (s *Service) Get(ctx context.Context, code, today string) (Plan, error) {
	code, e := currency.Code(code)
	if e != nil {
		return Plan{}, field("currencyCode", "Выберите валюту")
	}
	p, points, e := s.repo.Load(ctx, code)
	if e != nil || !p.Found {
		return p, e
	}
	factors, e := Resolve(p.Parameters, points, today)
	if e != nil {
		return p, e
	}
	p.Current, e = Calculate(p.Parameters, p.Initial, factors)
	return p, e
}
