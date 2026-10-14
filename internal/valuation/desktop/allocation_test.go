package desktop

import (
	"context"
	"errors"
	"myinvest/internal/valuation"
	"strings"
	"testing"
)

type allocationRepo struct {
	err   error
	calls int
}

func (r *allocationRepo) LoadAllocation(context.Context, string, string) (valuation.AllocationSnapshot, error) {
	r.calls++
	return valuation.AllocationSnapshot{}, r.err
}
func (r *allocationRepo) SaveAllocation(context.Context, string, valuation.AllocationPlan) error {
	r.calls++
	return r.err
}
func (r *allocationRepo) DeleteAllocation(context.Context, string, string, string) error {
	r.calls++
	return r.err
}
func TestAllocationAPIErrors(t *testing.T) {
	r := &allocationRepo{}
	api := NewAllocationAPI(func(fn func(*valuation.AllocationService) error) error { return fn(valuation.NewAllocationService(r)) })
	if _, e := api.GetAllocation("0", "2026-10-14"); e == nil || r.calls != 0 {
		t.Fatal("invalid pid hit storage", e)
	}
	if _, e := api.GetAllocation("1", "2026-02-29"); e == nil || r.calls != 0 {
		t.Fatal("invalid date hit storage", e)
	}
	out := api.SaveAllocation("1", valuation.AllocationPlan{CashPercent: "99"})
	if out.Errors["total"] == "" || r.calls != 0 {
		t.Fatal(out)
	}
	for _, err := range []error{valuation.ErrAllocationConflict, errors.New("private SQL data"), &valuation.AllocationFieldError{Field: "targets.0.assetID", Message: "Актив не найден"}} {
		r.err = err
		out = api.SaveAllocation("1", valuation.AllocationPlan{CashPercent: "100"})
		if strings.Contains(out.Message, "private") {
			t.Fatal(out)
		}
		if errors.Is(err, valuation.ErrAllocationConflict) && !out.Conflict {
			t.Fatal(out)
		}
		if field, ok := err.(*valuation.AllocationFieldError); ok && out.Errors[field.Field] != field.Message {
			t.Fatal(out)
		}
	}
	r.err = errors.New("private SQL data")
	if _, e := api.GetAllocation("1", "2026-10-14"); e == nil || strings.Contains(e.Error(), "private") {
		t.Fatal(e)
	}
	before := r.calls
	out = api.DeleteAllocation("1", "", "1")
	if !out.Conflict || before != r.calls {
		t.Fatal(out)
	}
}
