package portfolio

import (
	"context"
	"errors"
	"testing"
)

type stubRepository struct {
	current             Portfolio
	currency, duplicate bool
	writes              int
	except              string
	err                 error
}

func (r *stubRepository) List(context.Context) ([]Portfolio, error) {
	return []Portfolio{r.current}, r.err
}
func (r *stubRepository) Find(context.Context, string) (Portfolio, error) { return r.current, r.err }
func (r *stubRepository) CurrencyExists(context.Context, string) (bool, error) {
	return r.currency, r.err
}
func (r *stubRepository) NameExists(_ context.Context, _, except string) (bool, error) {
	r.except = except
	return r.duplicate, r.err
}
func (r *stubRepository) Create(_ context.Context, p Portfolio) (Portfolio, error) {
	r.writes++
	return p, r.err
}
func (r *stubRepository) Update(_ context.Context, p Portfolio) (Portfolio, error) {
	r.writes++
	return p, r.err
}
func TestServiceValidationBeforeWrite(t *testing.T) {
	for _, tc := range []struct {
		name                string
		input               Input
		currency, duplicate bool
		field               string
	}{
		{"unknown currency", Input{"P", "2026-01-01", "XXX"}, false, false, "currencyCode"},
		{"duplicate", Input{"P", "2026-01-01", "USD"}, true, true, "name"},
		{"invalid date", Input{"P", "2026-02-30", "USD"}, true, false, "openedOn"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &stubRepository{currency: tc.currency, duplicate: tc.duplicate}
			_, err := NewService(r).Create(context.Background(), tc.input)
			var e *FieldError
			if !errors.As(err, &e) || e.Field != tc.field || r.writes != 0 {
				t.Fatalf("%v; writes=%d", err, r.writes)
			}
		})
	}
}
func TestUpdatePreservesIdentityAndCurrency(t *testing.T) {
	r := &stubRepository{current: Portfolio{ID: "42", CurrencyCode: "USD"}, currency: true}
	s := NewService(r)
	p, err := s.Update(context.Background(), "42", Input{" Updated ", "2024-02-29", "USD"})
	if err != nil || p.ID != "42" || p.CurrencyCode != "USD" || r.except != "42" || r.writes != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	_, err = s.Update(context.Background(), "42", Input{"Updated", "2024-02-29", "RUB"})
	var e *FieldError
	if !errors.As(err, &e) || e.Field != "currencyCode" || r.writes != 1 {
		t.Fatalf("%v, writes=%d", err, r.writes)
	}
	r.err = ErrNotFound
	if _, err = s.Update(context.Background(), "999", Input{"P", "2026-01-01", "USD"}); !errors.Is(err, ErrNotFound) || r.writes != 1 {
		t.Fatal("missing update created record")
	}
}
