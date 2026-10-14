package desktop

import (
	"errors"
	"myinvest/internal/planning"
	"testing"
)

func TestErrorsDoNotExposeStorageDetails(t *testing.T) {
	for _, tc := range []struct {
		err   error
		field string
	}{
		{errors.New("private /database/path secret SQL"), ""},
		{&planning.FieldError{Field: "years", Message: "Введите число лет"}, "years"},
	} {
		api := NewAPI(func(func(*planning.Service) error) error { return tc.err })
		for _, out := range []PlanningResult{api.GetFinancialPlan("1", "2026-10-14"), api.PreviewFinancialPlan(planning.Parameters{}, "2026-10-14"), api.SaveFinancialPlan(planning.Parameters{}, "2026-10-14")} {
			if tc.field != "" {
				if out.Errors[tc.field] != "Введите число лет" || out.Message != "" {
					t.Fatal(out)
				}
			} else if out.Message != "Не удалось обработать финансовый план. Повторите попытку" || len(out.Errors) != 0 {
				t.Fatal(out)
			}
		}
	}
}
