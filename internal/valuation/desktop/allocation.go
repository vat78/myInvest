package desktop

import (
	"context"
	"errors"
	"myinvest/internal/valuation"
)

type AllocationAPI struct {
	withService func(func(*valuation.AllocationService) error) error
}

func NewAllocationAPI(fn func(func(*valuation.AllocationService) error) error) *AllocationAPI {
	return &AllocationAPI{fn}
}

type AllocationResult struct {
	Errors   map[string]string `json:"errors"`
	Message  string            `json:"message"`
	Conflict bool              `json:"conflict"`
}

func (a *AllocationAPI) GetAllocation(pid, date string) (valuation.Allocation, error) {
	if err := valuation.ValidatePortfolioID(pid); err != nil {
		return valuation.Allocation{}, err
	}
	if err := valuation.ValidateDate(date); err != nil {
		return valuation.Allocation{}, err
	}
	var out valuation.Allocation
	err := a.withService(func(s *valuation.AllocationService) error {
		var err error
		out, err = s.Load(context.Background(), pid, date)
		return err
	})
	if err != nil {
		return valuation.Allocation{}, errors.New("Не удалось загрузить целевое распределение. Повторите попытку")
	}
	return out, nil
}
func allocationResult(err error) AllocationResult {
	out := AllocationResult{Errors: map[string]string{}}
	var field *valuation.AllocationFieldError
	if errors.As(err, &field) {
		out.Errors[field.Field] = field.Message
	} else if errors.Is(err, valuation.ErrAllocationConflict) {
		out.Conflict = true
		out.Message = err.Error()
	} else if err != nil {
		out.Message = "Не удалось сохранить распределение. Повторите попытку"
	}
	return out
}
func (a *AllocationAPI) SaveAllocation(pid string, p valuation.AllocationPlan) AllocationResult {
	return allocationResult(a.withService(func(s *valuation.AllocationService) error { return s.Save(context.Background(), pid, p) }))
}
func (a *AllocationAPI) DeleteAllocation(pid, generation, revision string) AllocationResult {
	return allocationResult(a.withService(func(s *valuation.AllocationService) error {
		return s.Delete(context.Background(), pid, generation, revision)
	}))
}
func (a *AllocationAPI) PreviewAllocation(p valuation.AllocationPlan) valuation.AllocationPreview {
	return valuation.PreviewAllocation(p)
}
