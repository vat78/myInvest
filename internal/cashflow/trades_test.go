package cashflow

import "testing"

func TestAveragePrices(t *testing.T) {
	for _, tc := range []struct {
		amount, quantity, want string
		rounded                bool
	}{
		{"250", "2,5", "100", false}, {"1", "3", "0.333333333333333333", true},
		{"1", "6", "0.166666666666666667", true}, {"0.0000000000000000005", "1", "0.000000000000000001", true},
		{"0.00000000000000000049", "1", "0", true}, {"0.1", "0.2", "0.5", false},
		{"123456789012345678901", "0.000000000000000000001", "123456789012345678901000000000000000000000", false},
	} {
		price, rounded, err := AveragePrice(tc.amount, tc.quantity)
		if err != nil || price != tc.want || rounded != tc.rounded {
			t.Fatalf("%+v: %s %v %v", tc, price, rounded, err)
		}
	}
}
func TestTradeValidationAndTickers(t *testing.T) {
	base := Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "2,5", Amount: "250", Expenses: "0", CurrencyCode: "fake", PortfolioAmount: "fake"}
	got, err := ValidateTrade(base)
	if err != nil || got.Quantity != "2.5" || got.Expenses != "0" || got.CurrencyCode != "" || got.PortfolioAmount != "" {
		t.Fatal(got, err)
	}
	for _, change := range []func(*Input){func(i *Input) { i.Kind = "deposit" }, func(i *Input) { i.Date = "2026-02-30" }, func(i *Input) { i.Date = "0000-01-01" }, func(i *Input) { i.AssetID = "" }, func(i *Input) { i.Quantity = "0" }, func(i *Input) { i.Quantity = "-1" }, func(i *Input) { i.Amount = "0" }, func(i *Input) { i.Expenses = "-0.1" }, func(i *Input) { i.Expenses = "1e2" }} {
		in := base
		change(&in)
		if _, err := ValidateTrade(in); err == nil {
			t.Fatal(in)
		}
	}
	for _, tc := range []struct{ in, want string }{{" sber ", "SBER"}, {"brk.b", "BRK.B"}, {"123-@", "123-@"}, {"доля", "ДОЛЯ"}} {
		got, err := Ticker(tc.in)
		if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
	}
	for _, v := range []string{"", " \t", "BRK B", "A\nB", "A\x00"} {
		if _, err := Ticker(v); err == nil {
			t.Fatal(v)
		}
	}
}
func TestTradeMoneyAndHistory(t *testing.T) {
	for _, tc := range []struct{ kind, amount, expenses, want string }{{"buy", "250", "5", "-255"}, {"sell", "120", "2", "118"}, {"sell", "1", "2", "-1"}, {"sell", "0.1", "0.2", "-0.1"}, {"buy", "0.1", "0", "-0.1"}, {"sell", "1", "1", "0"}} {
		o := Operation{Input: Input{Kind: tc.kind, Amount: tc.amount, Expenses: tc.expenses}}
		if got := MoneyChange(o); got != tc.want {
			t.Fatal(tc, got)
		}
	}
	ops := []Operation{
		{ID: "10", Input: Input{Kind: "sell", Date: "2026-01-02", AssetID: "1", Quantity: "1", Amount: "120", Expenses: "2"}, Ticker: "SBER"},
		{ID: "2", Input: Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "2.5", Amount: "250", Expenses: "5"}, Ticker: "SBER"},
		{ID: "1", Input: Input{Kind: "deposit", Date: "2026-01-01", PortfolioAmount: "1000"}},
	}
	h, err := Rebuild(ops)
	if err != nil || h.Balance != "863" || len(h.Days) != 2 || h.Days[0].Balance != "745" || h.Days[1].Quantities["1"] != "1.5" || len(h.Warnings) != 0 {
		t.Fatal(h, err)
	}
	// ID 2 must precede ID 10 within the same date, and a sale after editing an
	// earlier buy must warn using its historical available quantity.
	ops[0].Date = "2026-01-01"
	ops[1].Quantity = "0.2"
	h, err = Rebuild(ops)
	if err != nil || len(h.Days) != 1 || h.Days[0].Quantities["1"] != "-0.8" || len(h.Warnings) != 1 || h.Warnings[0].Available != "0.2" {
		t.Fatal(h, err)
	}
	ops[0].Date = "2025-01-01"
	h, err = Rebuild(ops)
	if err != nil || h.Warnings[0].Available != "0" {
		t.Fatal(h, err)
	}
	h, err = Rebuild(nil)
	if err != nil || h.Balance != "0" || len(h.Days) != 0 {
		t.Fatal(h, err)
	}
}
func TestDailyFullPositionsAndExactFractions(t *testing.T) {
	ops := []Operation{
		{ID: "1", Input: Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "0.1", Amount: "0.1", Expenses: "0"}},
		{ID: "2", Input: Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "0.2", Amount: "0.2", Expenses: "0"}},
		{ID: "3", Input: Input{Kind: "sell", Date: "2026-01-02", AssetID: "1", Quantity: "0.3", Amount: "0.4", Expenses: "0.000000000000000000001"}},
		{ID: "4", Input: Input{Kind: "buy", Date: "9999-12-31", AssetID: "2", Quantity: "1", Amount: "1", Expenses: "0"}},
	}
	h, err := Rebuild(ops)
	if err != nil || len(h.Days) != 3 || h.Days[0].Quantities["1"] != "0.3" || h.Days[2].Quantities["1"] != "0" || h.Days[2].Quantities["2"] != "1" || h.Balance != "-0.900000000000000000001" {
		t.Fatal(h, err)
	}
}

func TestHistoricalCashWarningsRemainAfterRecovery(t *testing.T) {
	h, err := Rebuild([]Operation{
		{ID: "1", Input: Input{Kind: "withdrawal", Date: "2026-01-01", PortfolioAmount: "0.2"}},
		{ID: "2", Input: Input{Kind: "deposit", Date: "2026-01-02", PortfolioAmount: "1"}},
	})
	if err != nil || h.Balance != "0.8" || len(h.CashWarnings) != 1 || h.CashWarnings[0].Date != "2026-01-01" || h.CashWarnings[0].Balance != "-0.2" {
		t.Fatal(h, err)
	}
}
