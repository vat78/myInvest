package valuation

import (
	"fmt"
	"myinvest/internal/quotes"
	"reflect"
	"strings"
	"testing"
)

func TestOverviewValuationEventsAndUnchangedIntervals(t *testing.T) {
	s := Snapshot{Portfolios: []Portfolio{{ID: "1", Name: "P", Currency: "RUB", Days: []Day{
		{Date: "2026-01-01", Cash: "800", Deposits: "1000", Positions: []Position{{AssetID: "A", Ticker: "A", TypeCode: "stocks", TradeDate: "2026-01-01", Quantity: "2", Numerator: "200", Denominator: "1"}}},
		{Date: "2026-01-10", Cash: "800", Deposits: "1000", Positions: []Position{{AssetID: "A", Ticker: "A", TypeCode: "stocks", TradeDate: "2026-01-01", Quantity: "4", Numerator: "200", Denominator: "1"}}},
		{Date: "2026-01-12", Cash: "650", Deposits: "1200", Positions: []Position{{AssetID: "A", Ticker: "A", TypeCode: "stocks", TradeDate: "2026-01-12", Quantity: "5", Numerator: "750", Denominator: "1"}}},
	}}}, Pricing: Pricing{
		Quotes: []quotes.Quote{{AssetID: "A", Date: "2026-01-03", CurrencyCode: "USD", Price: "1.5"}, {AssetID: "A", Date: "2026-01-14", CurrencyCode: "RUB", Price: "160"}, {AssetID: "A", Date: "2026-01-17", CurrencyCode: "RUB", Price: "999"}},
		FX:     []ExchangeRate{{"2026-01-05", "USD", "RUB", "100"}, {"2026-01-08", "USD", "RUB", "110"}},
		Splits: []PriceSplit{{"1", "A", "2026-01-10", "1", "2"}},
	}}
	want := []string{"1000", "1000", "", "", "1100", "1100", "1100", "1130", "1130", "1130", "1130", "1400", "1400", "1450", "1450", "1450"}
	out, err := Build(s, "2026-01-16")
	if err != nil {
		t.Fatal(err)
	}
	for i, point := range out.Points {
		if point.Value != want[i] || point.Complete != (want[i] != "") {
			t.Fatalf("day %s: %#v want %s", point.Date, point, want[i])
		}
		if !point.Complete && (len(point.Issues) != 1 || !strings.Contains(point.Issues[0], "на "+point.Date+" ")) {
			t.Fatalf("issue date: %#v", point)
		}
		deposit := "1000"
		if i >= 11 {
			deposit = "1200"
		}
		if point.Deposits != deposit || !point.DepositsComplete {
			t.Fatalf("deposits: %#v", point)
		}
	}
	if out.Total.Value != "1450" || out.Portfolios[0].Summary.Value != "1450" {
		t.Fatal(out)
	}
	again, err := Build(s, "2026-01-16")
	if err != nil || !reflect.DeepEqual(out, again) {
		t.Fatalf("repeated build changed exact prices: %v", err)
	}
}

func BenchmarkOverviewSparseHistory(b *testing.B) {
	positions := make([]Position, 100)
	for i := range positions {
		positions[i] = Position{AssetID: fmt.Sprint(i + 1), TradeDate: "2016-01-01", Ticker: fmt.Sprintf("A%d", i), TypeCode: "stocks", Quantity: "10.123456789012345678", Numerator: "12345678901234567890123456789", Denominator: "1000000000000000000"}
	}
	s := Snapshot{Portfolios: []Portfolio{{ID: "1", Name: "P", Currency: "RUB", Days: []Day{{Date: "2016-01-01", Cash: "1000", Deposits: "1000", Positions: positions}}}}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Build(s, "2026-10-14"); err != nil {
			b.Fatal(err)
		}
	}
}
