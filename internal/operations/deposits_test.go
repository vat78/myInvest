package operations

import "testing"

func TestDepositTotals(t *testing.T) {
	for _, tc := range []struct {
		name, currency string
		ops            []Operation
		want           map[string]string
	}{
		{"RUB portfolio", "RUB", []Operation{
			{Input: Input{Kind: "deposit", Date: "2026-01-02", CurrencyCode: "USD", Amount: "10", PortfolioAmount: "900.1234567890123456789"}},
			{Input: Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "0.1", PortfolioAmount: "0.1"}},
			{Input: Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "0.2", PortfolioAmount: "0.2"}},
			{Input: Input{Kind: "withdrawal", Date: "2026-01-03", PortfolioAmount: "800"}},
			{Input: Input{Kind: "buy", Date: "2026-01-04", Amount: "12", Expenses: "1"}},
		}, map[string]string{"2026-01-01": "0.3", "2026-01-02": "900.4234567890123456789", "2026-01-03": "900.4234567890123456789", "2026-01-04": "900.4234567890123456789"}},
		{"RUB operation", "USD", []Operation{{Input: Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "94500", PortfolioAmount: "1000"}}}, map[string]string{"2026-01-01": "94500"}},
		{"missing RUB", "USD", []Operation{{Input: Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "USD", Amount: "100", PortfolioAmount: "100"}}, {Input: Input{Kind: "deposit", Date: "2026-01-02", CurrencyCode: "RUB", Amount: "1000", PortfolioAmount: "10"}}}, map[string]string{"2026-01-01": "", "2026-01-02": ""}},
		{"no deposits", "USD", []Operation{{Input: Input{Kind: "sell", Date: "2026-01-01"}}}, map[string]string{"2026-01-01": "0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DepositTotals(tc.ops, tc.currency)
			if err != nil {
				t.Fatal(err)
			}
			for day, want := range tc.want {
				if got[day] != want {
					t.Fatalf("%s: %s != %s", day, got[day], want)
				}
			}
		})
	}
	for _, value := range []string{"0", "-1", "invalid"} {
		if _, err := DepositTotals([]Operation{{Input: Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: value}}}, "USD"); err == nil {
			t.Fatal("accepted", value)
		}
	}
}
