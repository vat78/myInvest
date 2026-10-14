package valuation

import (
	"math/big"
	"myinvest/internal/operations"
	"myinvest/internal/quotes"
	"testing"
)

func TestPriceSelection(t *testing.T) {
	q := func(date, code, price string) quotes.Quote {
		return quotes.Quote{AssetID: "A", Date: date, CurrencyCode: code, Price: price}
	}
	tests := []struct {
		name, date, tradeDate, want, source, sourceDate string
		trade                                           *big.Rat
		pricing                                         Pricing
		incomplete                                      bool
	}{
		{name: "newer quote", date: "2026-01-03", tradeDate: "2026-01-01", trade: big.NewRat(100, 1), pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "RUB", "120")}}, want: "120", source: "quote", sourceDate: "2026-01-02"},
		{name: "newer trade", date: "2026-01-03", tradeDate: "2026-01-03", trade: big.NewRat(130, 1), pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "RUB", "120")}}, want: "130", source: "trade", sourceDate: "2026-01-03"},
		{name: "same day quote wins", date: "2026-01-02", tradeDate: "2026-01-02", trade: big.NewRat(100, 1), pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "RUB", "120")}}, want: "120", source: "quote", sourceDate: "2026-01-02"},
		{name: "future quote excluded", date: "2026-01-02", tradeDate: "2026-01-01", trade: big.NewRat(100, 1), pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-03", "RUB", "999")}}, want: "100", source: "trade", sourceDate: "2026-01-01"},
		{name: "exact FX on valuation date", date: "2026-01-03", tradeDate: "2026-01-01", trade: big.NewRat(100, 1), pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "USD", "0.1")}, FX: []ExchangeRate{{"2026-01-01", "USD", "RUB", "90"}, {"2026-01-03", "USD", "RUB", "95.1234567890123456789"}, {"2026-01-04", "USD", "RUB", "999"}}}, want: "9.51234567890123456789", source: "quote", sourceDate: "2026-01-02"},
		{name: "missing direct FX never selects older trade", date: "2026-01-03", tradeDate: "2026-01-01", trade: big.NewRat(100, 1), pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "USD", "2")}, FX: []ExchangeRate{{"2026-01-04", "USD", "RUB", "90"}, {"2026-01-01", "RUB", "USD", "0.01"}}}, source: "quote", sourceDate: "2026-01-02", incomplete: true},
		{name: "same day currencies prefer portfolio", date: "2026-01-03", pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "USD", "2"), q("2026-01-02", "RUB", "125"), q("2026-01-02", "EUR", "3")}}, want: "125", source: "quote", sourceDate: "2026-01-02"},
		{name: "different currencies lexical tie", date: "2026-01-03", pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "USD", "2"), q("2026-01-02", "EUR", "3")}, FX: []ExchangeRate{{"2026-01-02", "EUR", "RUB", "100"}}}, want: "300", source: "quote", sourceDate: "2026-01-02"},
		{name: "quote before split is adjusted", date: "2026-01-03", pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-01", "RUB", "1")}, Splits: []PriceSplit{{"1", "A", "2026-01-02", "1", "3"}}}, want: "1/3", source: "quote", sourceDate: "2026-01-01"},
		{name: "same day split does not adjust quote", date: "2026-01-03", pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-02", "RUB", "1")}, Splits: []PriceSplit{{"1", "A", "2026-01-02", "1", "3"}}}, want: "1", source: "quote", sourceDate: "2026-01-02"},
		{name: "other portfolio split is irrelevant", date: "2026-01-03", pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-01", "RUB", "100")}, Splits: []PriceSplit{{"2", "A", "2026-01-02", "1", "3"}}}, want: "100", source: "quote", sourceDate: "2026-01-01"},
		{name: "quote before reverse split", date: "2026-01-03", pricing: Pricing{Quotes: []quotes.Quote{q("2026-01-01", "RUB", "0.1")}, Splits: []PriceSplit{{"1", "A", "2026-01-02", "3", "1"}}}, want: "0.3", source: "quote", sourceDate: "2026-01-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := preparePrices(tt.pricing)
			if err != nil {
				t.Fatal(err)
			}
			chosen, err := idx.choose("1", "A", "RUB", tt.date, tt.tradeDate, tt.trade)
			if err != nil {
				t.Fatal(err)
			}
			if chosen.Source != tt.source || chosen.Date != tt.sourceDate {
				t.Fatalf("source: %#v", chosen)
			}
			if tt.incomplete {
				if chosen.value != nil || chosen.issue == "" {
					t.Fatalf("missing FX: %#v", chosen)
				}
				return
			}
			expected, ok := new(big.Rat).SetString(tt.want)
			if !ok || chosen.value == nil || chosen.value.Cmp(expected) != 0 {
				t.Fatalf("price: %#v want %s", chosen, tt.want)
			}
		})
	}
}
func TestOverviewQuotesOnDaysWithoutOperations(t *testing.T) {
	s := Snapshot{Portfolios: []Portfolio{{ID: "1", Name: "P", Currency: "RUB", Days: []Day{{Date: "2026-01-01", Cash: "800", Deposits: "1000", Positions: []Position{{AssetID: "A", TradeDate: "2026-01-01", Ticker: "A", TypeCode: "stocks", Quantity: "2", Numerator: "200", Denominator: "1"}}}}}}, Pricing: Pricing{Quotes: []quotes.Quote{
		{AssetID: "A", Date: "2026-01-02", CurrencyCode: "RUB", Price: "120"}, {AssetID: "A", Date: "2026-01-04", CurrencyCode: "USD", Price: "2"}, {AssetID: "A", Date: "2026-01-07", CurrencyCode: "RUB", Price: "999"},
	}, FX: []ExchangeRate{{"2026-01-05", "USD", "RUB", "100"}, {"2026-01-06", "USD", "RUB", "110"}}}}
	out, err := Build(s, "2026-01-06")
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"1000", "1040", "1040", "", "1200", "1240"} {
		p := out.Points[i]
		if p.Value != want || p.Complete != (want != "") || p.Deposits != "1000" {
			t.Fatalf("point %d: %#v", i, p)
		}
	}
	if out.Total.Value != "1240" || out.Portfolios[0].Summary.Groups[0].Value != "440" {
		t.Fatalf("current: %#v", out)
	}
	// Editing a historical quote changes only its and later applicable points.
	s.Pricing.Quotes[0].Price = "130"
	out, err = Build(s, "2026-01-06")
	if err != nil || out.Points[0].Value != "1000" || out.Points[1].Value != "1060" {
		t.Fatal(out, err)
	}
}

func TestAnalysisQuoteAndAverageAreIndependent(t *testing.T) {
	s := AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{analysisTrade("1", "buy", "A", "2", "200", "2026-01-01"), analysisTrade("2", "sell", "A", "1", "150", "2026-01-03")}, Pricing: Pricing{Quotes: []quotes.Quote{{AssetID: "A", Date: "2026-01-04", CurrencyCode: "USD", Price: "2"}}, FX: []ExchangeRate{{"2026-01-04", "USD", "RUB", "100"}}}}
	out, err := BuildAnalysis(s, "1", "2026-01-04")
	if err != nil || out.Total != "200" || out.Assets[0].AveragePrice != "100" || out.Assets[0].AverageValue != "100" || out.Assets[0].PriceSource != "quote" || out.Assets[0].PriceRate != "100" {
		t.Fatal(out, err)
	}
	s.Pricing.FX = nil
	out, err = BuildAnalysis(s, "1", "2026-01-04")
	if err != nil || out.Complete || out.Total != "" || out.Assets[0].AveragePrice != "100" || out.Assets[0].CurrentValue != "" {
		t.Fatal(out, err)
	}
	s.Pricing.Quotes[0].Date = "2026-01-02"
	out, err = BuildAnalysis(s, "1", "2026-01-04")
	if err != nil || out.Total != "150" || out.Assets[0].PriceSource != "trade" {
		t.Fatal(out, err)
	}
}
func TestAnalysisSplitQuoteAndZeroPosition(t *testing.T) {
	s := AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{analysisTrade("1", "buy", "A", "3", "1", "2026-01-01"), analysisSplit("2", "1", "3", "2026-01-03")}, Pricing: Pricing{Quotes: []quotes.Quote{{AssetID: "A", Date: "2026-01-02", CurrencyCode: "RUB", Price: "1"}}}}
	out, err := BuildAnalysis(s, "1", "2026-01-03")
	if err != nil || out.Total != "3" || out.Assets[0].Quantity != "9" || out.Assets[0].AverageValue != "1" {
		t.Fatal(out, err)
	}
	s.Operations = append(s.Operations, analysisTrade("3", "sell", "A", "9", "3", "2026-01-04"))
	s.Pricing.Quotes[0].CurrencyCode = "USD"
	s.Pricing.Quotes[0].Date = "2026-01-04"
	out, err = BuildAnalysis(s, "1", "2026-01-04")
	if err != nil || !out.Complete || out.Total != "0" || len(out.Assets) != 0 {
		t.Fatal(out, err)
	}
}
