package operations

import "testing"

func TestValidateDividend(t *testing.T) {
	for _, tt := range []struct{ amount, date, want string }{
		{"0,100", "2026-01-01", "0.1"}, {"00012.34000000000000000001", "9999-12-31", "12.34000000000000000001"},
		{"0", "2026-01-01", ""}, {"-1", "2026-01-01", ""}, {"1e3", "2026-01-01", ""}, {"", "2026-01-01", ""}, {"1", "2026-02-30", ""},
	} {
		for _, asset := range []string{"", "1"} {
			in, err := ValidateDividend(Input{Kind: "dividend", Date: tt.date, Amount: tt.amount, AssetID: asset, CurrencyCode: "USD", PortfolioAmount: "999", Quantity: "5", Expenses: "10", OldUnits: "2"})
			if tt.want == "" {
				if err == nil {
					t.Fatal("accepted", tt)
				}
				continue
			}
			if err != nil || in.Amount != tt.want || in.PortfolioAmount != tt.want || in.AssetID != asset || in.CurrencyCode != "" || in.Quantity != "" || in.Expenses != "" || in.OldUnits != "" {
				t.Fatal(in, err)
			}
		}
	}
}
func TestDividendExactHistory(t *testing.T) {
	ops := []Operation{
		{ID: "1", Input: Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "100", PortfolioAmount: "100"}},
		{ID: "2", Input: Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "0.5", Amount: "200", Expenses: "0"}},
		{ID: "3", Input: Input{Kind: "dividend", Date: "2026-01-02", AssetID: "1", Amount: "0.1", PortfolioAmount: "0.1"}},
		{ID: "4", Input: Input{Kind: "dividend", Date: "2026-01-02", AssetID: "2", Amount: "0.2", PortfolioAmount: "0.2"}},
	}
	h, err := Rebuild(ops)
	if err != nil || h.Balance != "-99.7" || h.Days[1].Quantities["1"] != "0.5" || len(h.Days[1].Quantities) != 1 || h.Days[1].Values["1"].PriceOperationID != "2" {
		t.Fatal(h, err)
	}
	totals, err := DepositTotals(ops, "RUB")
	if err != nil || totals["2026-01-02"] != "100" {
		t.Fatal(totals, err)
	}
}
