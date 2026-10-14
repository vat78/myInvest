package desktop

import (
	"context"
	"errors"
	"myinvest/internal/quotes"
	"strings"
	"testing"
)

type repository struct {
	saved quotes.Quote
	err   error
}

func (r *repository) List(context.Context) ([]quotes.Quote, error) {
	return []quotes.Quote{r.saved}, r.err
}
func (r *repository) Save(_ context.Context, q quotes.Quote) error { r.saved = q; return r.err }
func TestExactDTOAndErrors(t *testing.T) {
	r := &repository{}
	api := NewAPI(func(fn func(*quotes.Service) error) error { return fn(quotes.NewService(r)) })
	in := QuoteDTO{AssetID: "1", Date: "2026-10-14", CurrencyCode: "RUB", Price: "0.1234567890123456789012345678900", Ticker: "forged"}
	out := api.SaveQuote(in)
	if out.Message != "" || len(out.Errors) != 0 || r.saved.Price != in.Price || r.saved.Ticker != "" {
		t.Fatalf("save: %#v %#v", out, r.saved)
	}
	list, err := api.ListQuotes()
	if err != nil || list[0].Price != in.Price {
		t.Fatalf("list: %#v %v", list, err)
	}
	in.Price = "0"
	out = api.SaveQuote(in)
	if out.Errors["price"] == "" {
		t.Fatalf("validation: %#v", out)
	}
	r.err = errors.New("SQL private database path")
	in.Price = "1"
	out = api.SaveQuote(in)
	if out.Message == "" || strings.Contains(out.Message, "private") {
		t.Fatalf("storage: %#v", out)
	}
	_, err = api.ListQuotes()
	if err == nil || strings.Contains(err.Error(), "private") {
		t.Fatalf("read: %v", err)
	}
}
