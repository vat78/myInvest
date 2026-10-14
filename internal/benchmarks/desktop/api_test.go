package desktop

import (
	"errors"
	"myinvest/internal/benchmarks"
	"testing"
)

func TestSafeErrors(t *testing.T) {
	for _, tc := range []struct {
		err   error
		field string
	}{{errors.New("secret SQL"), ""}, {&benchmarks.FieldError{Field: "input", Message: "invalid"}, "input"}} {
		a := NewAPI(func(func(*benchmarks.Service) error) error { return tc.err })
		out := a.SaveBenchmarkPoint(benchmarks.Point{})
		if tc.field != "" {
			if out.Errors[tc.field] != "invalid" {
				t.Fatal(out)
			}
		} else if out.Message != "Не удалось сохранить данные. Повторите попытку" {
			t.Fatal(out)
		}
		if _, e := a.ListBenchmarks(); e == nil || e.Error() != "Не удалось загрузить бенчмарки. Повторите попытку" {
			t.Fatal(e)
		}
	}
}
