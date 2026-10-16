package valuation

import (
	"myinvest/internal/quotes"
	"strings"
	"testing"
)

func TestCapitalHistoryCurrencyAndDates(t *testing.T) {
	s := Snapshot{Portfolios: []Portfolio{
		{ID: "1", Name: "R", Currency: "RUB", Days: []Day{{Date: "2026-01-01", Cash: "100.125"}, {Date: "2026-01-04", Cash: "99999"}}},
		{ID: "2", Name: "U", Currency: "USD", Days: []Day{{Date: "2026-01-01", Cash: "-0.25"}}},
	}, Pricing: Pricing{FX: []ExchangeRate{
		{Date: "2026-01-02", Base: "RUB", Quote: "USD", Value: "0.01"},
		{Date: "2026-01-03", Base: "RUB", Quote: "USD", Value: "0.02"},
		{Date: "2026-01-04", Base: "RUB", Quote: "USD", Value: "100"},
		{Date: "2025-01-01", Base: "USD", Quote: "RUB", Value: "100"},
	}}}
	out, err := BuildCapitalHistory(s, "USD", "2025-12-31", "2026-01-03")
	if err != nil || len(out) != 3 {
		t.Fatal(out, err)
	}
	if out[0].Complete || out[0].Value != "" || !strings.Contains(out[0].Issues[0], "RUB → USD") {
		t.Fatal("missing direct rate must not use inverse or future rate", out[0])
	}
	for i, want := range []string{"0.75125", "1.7525"} {
		if !out[i+1].Complete || out[i+1].Value != want {
			t.Fatal(out[i+1], want)
		}
	}
	out, err = BuildCapitalHistory(s, "USD", "2026-01-03", "2026-01-03")
	if err != nil || len(out) != 1 || out[0].Value != "1.7525" {
		t.Fatal("start boundary", out, err)
	}
}

func TestCapitalHistoryQuotesSplitsAndGaps(t *testing.T) {
	s := Snapshot{Portfolios: []Portfolio{{ID: "1", Name: "P", Currency: "RUB", Days: []Day{
		{Date: "2026-01-01", Cash: "0", Positions: []Position{{AssetID: "1", Ticker: "A", Quantity: "1"}}},
		{Date: "2026-01-03", Cash: "0", Positions: []Position{{AssetID: "1", Ticker: "A", Quantity: "2"}}},
	}}}, Pricing: Pricing{
		Quotes: []quotes.Quote{{AssetID: "1", Date: "2026-01-02", CurrencyCode: "RUB", Price: "10.123456789012345678"}},
		Splits: []PriceSplit{{PortfolioID: "1", AssetID: "1", Date: "2026-01-03", OldUnits: "1", NewUnits: "2"}},
	}}
	out, err := BuildCapitalHistory(s, "RUB", "2026-01-01", "2026-01-04")
	if err != nil || len(out) != 4 || out[0].Complete || len(out[0].Issues) == 0 {
		t.Fatal(out, err)
	}
	for _, p := range out[1:] {
		if !p.Complete || p.Value != "10.123456789012345678" {
			t.Fatal("split or cached value changed exact capital", p)
		}
	}
}

func TestCapitalHistoryEmptyAndZero(t *testing.T) {
	for _, tc := range []struct{ start, end string }{{"2026-01-01", "2026-01-02"}, {"2027-01-01", "2026-01-01"}} {
		out, err := BuildCapitalHistory(Snapshot{}, "RUB", tc.start, tc.end)
		if err != nil || out == nil || len(out) != 0 {
			t.Fatal(out, err)
		}
	}
	s := Snapshot{Portfolios: []Portfolio{{Currency: "USD", Days: []Day{{Date: "2026-01-01", Cash: "0"}}}}}
	out, err := BuildCapitalHistory(s, "RUB", "2026-01-01", "2026-01-02")
	if err != nil || len(out) != 2 || !out[0].Complete || out[0].Value != "0" || out[1].Value != "0" {
		t.Fatal(out, err)
	}
	for _, tc := range []struct{ code, start, end string }{{"bad!", "2026-01-01", "2026-01-02"}, {"RUB", "2026-02-30", "2026-01-02"}, {"RUB", "2026-01-01", ""}} {
		if _, err := BuildCapitalHistory(s, tc.code, tc.start, tc.end); err == nil {
			t.Fatal(tc)
		}
	}
}
