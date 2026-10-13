package portfolio

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct{ name, date, code, wantName, wantKey, field string }{
		{" Основной ", "2024-02-29", "RUB", "Основной", "основной", ""},
		{"\u2003Growth  FUND\u00a0", "2099-12-31", "USD", "Growth  FUND", "growth  fund", ""},
		{"  ", "2026-01-01", "USD", "", "", "name"},
		{"\t\u2003", "2026-01-01", "USD", "", "", "name"},
		{"P", "", "USD", "", "", "openedOn"},
		{"P", "2025-02-29", "USD", "", "", "openedOn"},
		{"P", "2026-02-30", "USD", "", "", "openedOn"},
		{"P", "2026-1-01", "USD", "", "", "openedOn"},
		{"P", "0000-01-01", "USD", "", "", "openedOn"},
		{"P", "2026-01-01T00:00:00Z", "USD", "", "", "openedOn"},
		{"P", "2026-01-01", "", "", "", "currencyCode"},
	} {
		t.Run(tc.name+tc.date+tc.code, func(t *testing.T) {
			p, err := Validate(Input{tc.name, tc.date, tc.code})
			if tc.field != "" {
				var e *FieldError
				if !errors.As(err, &e) || e.Field != tc.field {
					t.Fatalf("error: %v", err)
				}
				return
			}
			if err != nil || p.Name != tc.wantName || p.NameKey != tc.wantKey || p.OpenedOn != tc.date || p.CurrencyCode != tc.code {
				t.Fatalf("got %+v, %v", p, err)
			}
		})
	}
}
