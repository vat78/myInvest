package valuation

import (
	"math/big"
	"myinvest/internal/operations"
	"myinvest/internal/quotes"
	"strings"
	"testing"
)

func allocationBuy(id, asset, q, amount, date string) operations.Operation {
	return operations.Operation{ID: id, PortfolioID: "1", Ticker: asset, Input: operations.Input{Kind: "buy", AssetID: asset, Quantity: q, Amount: amount, Expenses: "0", Date: date}}
}
func allocationDeposit(amount string) operations.Operation {
	return operations.Operation{ID: "1", PortfolioID: "1", Input: operations.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: amount, PortfolioAmount: amount}}
}
func allocationFixture(deposit string) AllocationSnapshot {
	return AllocationSnapshot{Analysis: AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{allocationDeposit(deposit), allocationBuy("2", "1", "6", "600", "2026-01-02"), allocationBuy("3", "2", "3", "300", "2026-01-02")}}, Assets: []operations.Asset{{ID: "1", Ticker: "A"}, {ID: "2", Ticker: "B"}}, Plan: AllocationPlan{Generation: "test", Revision: "1", CashPercent: "20", Targets: []Target{{"1", "40"}, {"2", "40"}}}}
}
func TestAllocationReserveScenarios(t *testing.T) {
	for _, tt := range []struct{ deposit, total, reserve, buy, remaining, shortfall, sell, purchase, target string }{
		{"1200", "1200", "240", "60", "240", "0", "-120", "180", "4.8"},
		{"1000", "1000", "200", "0", "100", "100", "-200", "100", "4"},
		{"1125", "1125", "225", "0", "225", "0", "-150", "150", "4.5"},
	} {
		t.Run(tt.deposit, func(t *testing.T) {
			out, err := BuildAllocation(allocationFixture(tt.deposit), "1", "2026-10-14")
			if err != nil || !out.CanCalculate || len(out.Rows) != 3 {
				t.Fatal(out, err)
			}
			a, b := out.Rows[0], out.Rows[1]
			if out.Total != tt.total || out.Reserve != tt.reserve || out.CashAfterBuys != tt.remaining || out.ReserveShortfall != tt.shortfall || a.BuyValue != "0" || b.BuyValue != tt.buy || a.RebalanceValue != tt.sell || b.RebalanceValue != tt.purchase || a.TargetQuantity != tt.target {
				t.Fatalf("%+v rows=%+v", out, out.Rows)
			}
		})
	}
}
func TestAllocationPercentValidation(t *testing.T) {
	for _, v := range []string{"-1", "100.01", "1.001", "1e1", "1/2", "", ".5", "1."} {
		if _, err := PercentUnits(v); err == nil {
			t.Errorf("accepted %q", v)
		}
	}
	if n, e := PercentUnits(" 012,34 "); e != nil || n != 1234 {
		t.Fatal(n, e)
	}
	for _, tt := range []struct {
		name string
		p    AllocationPlan
	}{
		{"sum", AllocationPlan{CashPercent: "99.99"}},
		{"duplicate", AllocationPlan{CashPercent: "0", Targets: []Target{{"1", "50"}, {"1", "50"}}}},
		{"id", AllocationPlan{CashPercent: "0", Targets: []Target{{"0", "100"}}}},
		{"precision", AllocationPlan{CashPercent: "100.000"}},
		{"version", AllocationPlan{CashPercent: "100", Generation: "test"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, e := ValidateAllocation(tt.p); e == nil {
				t.Fatal("accepted")
			}
		})
	}
	p, e := ValidateAllocation(AllocationPlan{CashPercent: "0", Targets: []Target{{"1", "12,34"}, {"2", "87.66"}}})
	if e != nil || p.Targets[0].Percent != "12.34" {
		t.Fatal(p, e)
	}
	preview := PreviewAllocation(AllocationPlan{CashPercent: "20", Targets: []Target{{"1", "81"}}})
	if preview.Total != "101" || preview.Difference != "-1" {
		t.Fatal(preview)
	}
}
func TestAllocationExceptionalStates(t *testing.T) {
	for _, tt := range []struct {
		name           string
		modify         func(*AllocationSnapshot)
		complete, calc bool
	}{
		{"missing held price", func(s *AllocationSnapshot) { s.Analysis.Operations[1].Kind = "sell" }, false, false},
		{"negative money", func(s *AllocationSnapshot) { s.Analysis.Operations[0] = allocationDeposit("1") }, true, false},
		{"negative position", func(s *AllocationSnapshot) {
			s.Analysis.Operations = append(s.Analysis.Operations, operations.Operation{ID: "4", PortfolioID: "1", Input: operations.Input{Kind: "sell", AssetID: "1", Quantity: "7", Amount: "700", Expenses: "0", Date: "2026-01-03"}})
		}, true, false},
		{"new unpriced target", func(s *AllocationSnapshot) {
			s.Plan.Targets[1].AssetID = "3"
			s.Assets = append(s.Assets, operations.Asset{ID: "3", Ticker: "C"})
		}, true, false},
		{"new quoted target", func(s *AllocationSnapshot) {
			s.Plan.Targets[1].AssetID = "3"
			s.Assets = append(s.Assets, operations.Asset{ID: "3", Ticker: "C"})
			s.Analysis.Pricing.Quotes = []quotes.Quote{{AssetID: "3", Date: "2026-01-03", CurrencyCode: "RUB", Price: "100"}}
		}, true, true},
		{"missing FX", func(s *AllocationSnapshot) {
			s.Analysis.Pricing.Quotes = []quotes.Quote{{AssetID: "1", Date: "2026-01-03", CurrencyCode: "USD", Price: "1"}}
		}, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := allocationFixture("1200")
			tt.modify(&s)
			out, e := BuildAllocation(s, "1", "2026-10-14")
			if e != nil || out.Complete != tt.complete || out.CanCalculate != tt.calc {
				t.Fatal(out, e)
			}
		})
	}
	s := allocationFixture("1200")
	s.Plan = AllocationPlan{Generation: "cash", Revision: "1", CashPercent: "100"}
	out, e := BuildAllocation(s, "1", "2026-10-14")
	if e != nil || out.Rows[0].RebalanceQuantity != "-6" || out.Reserve != "1200" {
		t.Fatal(out, e)
	}
	s.Analysis.Operations = nil
	out, e = BuildAllocation(s, "1", "2026-10-14")
	if e != nil || !out.CanCalculate || out.Total != "0" {
		t.Fatal(out, e)
	}
	s.Plan.Targets = []Target{{"1", "100"}}
	s.Plan.CashPercent = "0"
	out, e = BuildAllocation(s, "1", "2026-10-14")
	if e != nil || !out.CanCalculate || out.Rows[0].TargetQuantity != "0" {
		t.Fatal(out, e)
	}
}
func TestAllocationLatestQuoteOrPurchase(t *testing.T) {
	for _, tt := range []struct{ name, quoteDay, currency, expected, source string }{
		{"older quote", "2026-01-01", "RUB", "600", "trade"},
		{"same day quote", "2026-01-02", "RUB", "1200", "quote"},
		{"new quote", "2026-01-03", "RUB", "1200", "quote"},
		{"future quote", "2027-01-01", "RUB", "600", "trade"},
		{"foreign quote", "2026-01-03", "USD", "1200", "quote"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := allocationFixture("1200")
			price := "200"
			if tt.currency == "USD" {
				price = "2"
			}
			s.Analysis.Pricing = Pricing{Quotes: []quotes.Quote{{AssetID: "1", Date: tt.quoteDay, CurrencyCode: tt.currency, Price: price}}, FX: []ExchangeRate{{Date: "2026-01-03", Base: "USD", Quote: "RUB", Value: "100"}}}
			out, e := BuildAllocation(s, "1", "2026-10-14")
			if e != nil || out.Rows[0].CurrentValue != tt.expected || out.Rows[0].PriceSource != tt.source {
				t.Fatal(out, e)
			}
		})
	}
	s := allocationFixture("1200")
	sale := allocationBuy("4", "1", "1", "999", "2026-01-04")
	sale.Kind = "sell"
	s.Analysis.Operations = append(s.Analysis.Operations, sale)
	out, e := BuildAllocation(s, "1", "2026-10-14")
	if e != nil || out.Rows[0].CurrentValue != "500" || out.Rows[0].PriceDate != "2026-01-02" {
		t.Fatal(out, e)
	}
	// A split after the source adjusts price and quantity without changing value.
	s = allocationFixture("1200")
	s.Analysis.Operations = append(s.Analysis.Operations, operations.Operation{ID: "4", PortfolioID: "1", Input: operations.Input{Kind: "split", Date: "2026-01-03", AssetID: "1", OldUnits: "1", NewUnits: "3"}})
	out, e = BuildAllocation(s, "1", "2026-10-14")
	if e != nil || out.Rows[0].Quantity != "18" || out.Rows[0].TargetQuantity != "14.4" || out.Rows[0].CurrentValue != "600" {
		t.Fatal(out, e)
	}
	s.Analysis.Operations[0].PortfolioID = "2"
	if _, e = BuildAllocation(s, "1", "2026-10-14"); e == nil {
		t.Fatal("mixed portfolios")
	}
}
func TestAllocationFractionalBudgetsExpensesAndRounding(t *testing.T) {
	s := allocationFixture("1200")
	s.Analysis.Operations[1].Expenses = "0.1234567890123456789"
	out, e := BuildAllocation(s, "1", "2026-10-14")
	if e != nil || out.Cash != "≈299.88" {
		t.Fatal(out, e)
	}
	s = AllocationSnapshot{Analysis: AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{allocationDeposit("1")}, Pricing: Pricing{Quotes: []quotes.Quote{{AssetID: "1", Date: "2026-01-01", CurrencyCode: "RUB", Price: "3"}, {AssetID: "2", Date: "2026-01-01", CurrencyCode: "RUB", Price: "7"}}}}, Assets: []operations.Asset{{ID: "1", Ticker: "A"}, {ID: "2", Ticker: "B"}}, Plan: AllocationPlan{Generation: "a", Revision: "1", CashPercent: "0", Targets: []Target{{"1", "33.33"}, {"2", "66.67"}}}}
	out, e = BuildAllocation(s, "1", "2026-10-14")
	if e != nil || out.Rows[0].BuyQuantity != "0.1111" || out.Rows[0].BuyValue != "≈0.33" || !strings.HasPrefix(out.Rows[1].BuyQuantity, "≈") || out.CashAfterBuys != "0" {
		t.Fatal(out, e)
	}
	if roundedLabel(bigRat("0.00000000000000000001"), 18) != "≈+0" {
		t.Fatal("tiny rounded to zero")
	}
}
func bigRat(s string) *big.Rat { r, _ := new(big.Rat).SetString(s); return r }

func TestAllocationMultipleDeficits(t *testing.T) {
	s := AllocationSnapshot{Analysis: AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{allocationDeposit("10000"), allocationBuy("2", "1", "60", "6000", "2026-01-02"), allocationBuy("3", "2", "10", "1000", "2026-01-02"), allocationBuy("4", "3", "5", "500", "2026-01-02")}}, Assets: []operations.Asset{{ID: "1", Ticker: "A"}, {ID: "2", Ticker: "B"}, {ID: "3", Ticker: "C"}}, Plan: AllocationPlan{Generation: "g", Revision: "1", CashPercent: "10", Targets: []Target{{"1", "20"}, {"2", "30"}, {"3", "40"}}}}
	out, e := BuildAllocation(s, "1", "2026-10-14")
	if e != nil || out.Reserve != "1000" || out.CashAfterBuys != "1000" || out.Rows[0].BuyValue != "0" || out.Rows[1].BuyValue != "≈545.45" || out.Rows[2].BuyValue != "≈954.55" || out.Rows[0].RebalanceValue != "-4000" || out.Rows[1].RebalanceValue != "2000" || out.Rows[2].RebalanceValue != "3500" {
		t.Fatal(out, e)
	}
	// Future buys do not enter holdings, cash, or the purchase-price candidate.
	s.Analysis.Operations = append(s.Analysis.Operations, allocationBuy("5", "1", "1", "999999", "2027-01-01"))
	future, e := BuildAllocation(s, "1", "2026-10-14")
	if e != nil || future.Total != out.Total || future.Rows[0].Quantity != "60" {
		t.Fatal(future, e)
	}
}

func TestAllocationExactPurchasePrice(t *testing.T) {
	for _, tt := range []struct {
		name, quantity, amount, deposit, total, reserve, target, targetQuantity, buyQuantity, buyValue string
		split                                                                                          bool
	}{
		{"repeating price", "3", "100", "200", "200", "50", "150", "4.5", "1.5", "50", false},
		{"rounding changes valuation", "3000000000000000000", "100", "200", "200", "50", "150", "4500000000000000000", "1500000000000000000", "50", false},
		{"positive price below display precision", "3000000000000000000", "1", "2", "2", "0.5", "1.5", "4500000000000000000", "1500000000000000000", "0.5", false},
		{"split adjusts exact price", "3", "100", "200", "200", "50", "150", "13.5", "4.5", "50", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := AllocationSnapshot{
				Analysis: AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{
					allocationDeposit(tt.deposit), allocationBuy("2", "1", tt.quantity, tt.amount, "2026-01-02"),
				}},
				Assets: []operations.Asset{{ID: "1", Ticker: "A"}},
				Plan:   AllocationPlan{Generation: "g", Revision: "1", CashPercent: "25", Targets: []Target{{"1", "75"}}},
			}
			if tt.split {
				s.Analysis.Operations = append(s.Analysis.Operations, operations.Operation{ID: "3", PortfolioID: "1", Input: operations.Input{Kind: "split", Date: "2026-01-03", AssetID: "1", OldUnits: "1", NewUnits: "3"}})
			}
			out, err := BuildAllocation(s, "1", "2026-10-14")
			if err != nil {
				t.Fatal(err)
			}
			if !out.Complete || !out.CanCalculate || len(out.Rows) != 2 {
				t.Fatalf("allocation unavailable: %+v", out)
			}
			row := out.Rows[0]
			if row.CurrentValue != tt.amount || out.Total != tt.total || out.Reserve != tt.reserve || out.CashAfterBuys != tt.reserve || out.ReserveShortfall != "0" || row.TargetValue != tt.target || row.TargetQuantity != tt.targetQuantity || row.BuyQuantity != tt.buyQuantity || row.RebalanceQuantity != tt.buyQuantity || row.BuyValue != tt.buyValue || row.RebalanceValue != tt.buyValue {
				t.Fatalf("allocation=%+v row=%+v", out, row)
			}
		})
	}
}

func TestDividendsIncreaseAllocationCash(t *testing.T) {
	s := allocationFixture("1200")
	before, err := BuildAllocation(s, "1", "2026-10-14")
	if err != nil {
		t.Fatal(err)
	}
	s.Analysis.Operations = append(s.Analysis.Operations, operations.Operation{ID: "4", PortfolioID: "1", Input: operations.Input{Kind: "dividend", Date: "2026-01-03", Amount: "0.3", PortfolioAmount: "0.3"}})
	after, err := BuildAllocation(s, "1", "2026-10-14")
	if err != nil || before.Cash != "300" || after.Cash != "300.3" {
		t.Fatal(before, after, err)
	}
}
