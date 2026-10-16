package desktop

import (
	"context"
	"errors"
	"myinvest/internal/planning"
)

type PlanningResult struct {
	Plan    planning.Plan     `json:"plan"`
	Errors  map[string]string `json:"errors"`
	Message string            `json:"message"`
}
type PlanningAPI struct {
	useService func(func(*planning.Service) error) error
}

func NewAPI(fn func(func(*planning.Service) error) error) *PlanningAPI { return &PlanningAPI{fn} }
func (a *PlanningAPI) call(fn func(*planning.Service) (planning.Plan, error)) PlanningResult {
	out := PlanningResult{Errors: map[string]string{}}
	e := a.useService(func(s *planning.Service) error { var e error; out.Plan, e = fn(s); return e })
	var f *planning.FieldError
	if errors.As(e, &f) {
		out.Errors[f.Field] = f.Message
	} else if e != nil {
		out.Message = "Не удалось обработать финансовый план. Повторите попытку"
	}
	return out
}
func (a *PlanningAPI) GetFinancialPlan(code, date string) PlanningResult {
	return a.call(func(s *planning.Service) (planning.Plan, error) { return s.Get(context.Background(), code, date) })
}
func (a *PlanningAPI) PreviewFinancialPlan(p planning.Parameters, date string) PlanningResult {
	return a.call(func(s *planning.Service) (planning.Plan, error) { return s.Preview(context.Background(), p, date) })
}
func (a *PlanningAPI) SaveFinancialPlan(p planning.Parameters, date string) PlanningResult {
	return a.call(func(s *planning.Service) (planning.Plan, error) { return s.Save(context.Background(), p, date) })
}
