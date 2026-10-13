package currency

import (
	"context"
	"errors"
	"testing"
)

type memoryRepository struct {
	codes  []string
	rates  map[Pair]string
	writes int
	err    error
}

func (r *memoryRepository) ListCurrencies(context.Context) ([]string, error) { return r.codes, r.err }
func (r *memoryRepository) AddCurrency(_ context.Context, code string) error {
	if r.err != nil {
		return r.err
	}
	r.codes = append(r.codes, code)
	r.writes++
	return nil
}
func (r *memoryRepository) FindRate(_ context.Context, pair Pair) (string, bool, error) {
	if r.err != nil {
		return "", false, r.err
	}
	value, found := r.rates[pair]
	return value, found, nil
}
func (r *memoryRepository) SaveRates(_ context.Context, direct, inverse Rate) error {
	if r.err != nil {
		return r.err
	}
	if r.rates == nil {
		r.rates = make(map[Pair]string)
	}
	r.rates[direct.Pair], r.rates[inverse.Pair] = direct.Value, inverse.Value
	r.writes++
	return nil
}

func TestServiceNormalizesAndPreservesRate(t *testing.T) {
	repo := &memoryRepository{}
	s := NewService(repo)
	ctx := context.Background()
	if code, err := s.Add(ctx, " eur "); err != nil || code != "EUR" {
		t.Fatalf("add: %q %v", code, err)
	}
	if codes, err := s.List(ctx); err != nil || len(codes) != 1 || codes[0] != "EUR" {
		t.Fatalf("list: %v %v", codes, err)
	}
	if err := s.Save(ctx, "2026-10-12", "eur", " rub ", "00095,12345678901234567890123456789000"); err != nil {
		t.Fatal(err)
	}
	if value, found, err := s.Find(ctx, "2026-10-12", " EUR ", "rub"); err != nil || !found || value != "95.12345678901234567890123456789000" {
		t.Fatalf("find: %q %v %v", value, found, err)
	}
	if _, found, err := s.Find(ctx, "2026-10-13", "EUR", "RUB"); err != nil || found {
		t.Fatal("carried forward rate")
	}
	if value, found, err := s.Find(ctx, "2026-10-12", "RUB", "EUR"); err != nil || !found || value != "0.010512654120823639" {
		t.Fatalf("inverse: %q %v %v", value, found, err)
	}
}

func TestServiceRejectsInvalidInputBeforeStorage(t *testing.T) {
	repo := &memoryRepository{}
	s := NewService(repo)
	ctx := context.Background()
	if _, err := s.Add(ctx, "bad!"); err == nil {
		t.Fatal("invalid code accepted")
	}
	for _, input := range []struct{ date, base, quote, value string }{
		{"2026-02-30", "USD", "RUB", "95"}, {"2026-10-12", "USD", "usd", "95"},
		{"2026-10-12", "USD", "RUB", "4000000000000000000"},
		{"2026-10-12", "USD", "RUB", "0"}, {"2026-10-12", "USD", "RUB", "-95"},
	} {
		if err := s.Save(ctx, input.date, input.base, input.quote, input.value); err == nil {
			t.Fatalf("accepted: %+v", input)
		}
	}
	if _, _, err := s.Find(ctx, "bad date", "USD", "RUB"); err == nil {
		t.Fatal("invalid query accepted")
	}
	if repo.writes != 0 {
		t.Fatal("invalid input reached storage")
	}
}

func TestServicePropagatesStorageFailures(t *testing.T) {
	failure := errors.New("storage unavailable")
	s := NewService(&memoryRepository{err: failure})
	ctx := context.Background()
	_, listErr := s.List(ctx)
	_, addErr := s.Add(ctx, "EUR")
	_, _, findErr := s.Find(ctx, "2026-10-12", "USD", "RUB")
	saveErr := s.Save(ctx, "2026-10-12", "USD", "RUB", "95")
	for _, err := range []error{listErr, addErr, findErr, saveErr} {
		if !errors.Is(err, failure) {
			t.Fatalf("lost repository error: %v", err)
		}
	}
}
