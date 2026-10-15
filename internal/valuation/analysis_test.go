package valuation

import (
	"myinvest/internal/operations"
	"strings"
	"testing"
)

func analysisTrade(id, kind, asset, q, amount, date string) operations.Operation {
	return operations.Operation{ID: id, PortfolioID: "1", Ticker: asset, Input: operations.Input{Kind: kind, Date: date, AssetID: asset, Quantity: q, Amount: amount, Expenses: "999"}}
}
func analysisSplit(id, old, next, date string) operations.Operation {
	return operations.Operation{ID: id, PortfolioID: "1", Ticker: "A", Input: operations.Input{Kind: "split", Date: date, AssetID: "A", OldUnits: old, NewUnits: next}}
}
func TestAnalysisFIFORules(t *testing.T) {
	tests := []struct {
		name                          string
		ops                           []operations.Operation
		quantity, current, avg, value string
	}{
		{"sales consume oldest purchase, latest sell sets current price", []operations.Operation{analysisTrade("1", "buy", "A", "2", "200", "2026-01-01"), analysisTrade("2", "buy", "A", "1", "160", "2026-01-02"), analysisTrade("3", "sell", "A", "1", "900", "2026-01-03")}, "2", "1800", "130", "260"},
		{"sale crosses purchase lots", []operations.Operation{analysisTrade("1", "buy", "A", "10", "1000", "2026-01-01"), analysisTrade("2", "buy", "A", "10", "2000", "2026-01-02"), analysisTrade("3", "sell", "A", "12", "3600", "2026-01-03")}, "8", "2400", "200", "1600"},
		{"fractional FIFO without intermediate rounding", []operations.Operation{analysisTrade("1", "buy", "A", "0.3", "1", "2026-01-01"), analysisTrade("2", "sell", "A", "0.1", "10", "2026-01-02")}, "0.2", "20", "3.33", "0.67"},
		{"numeric ID FIFO on same day", []operations.Operation{analysisTrade("10", "buy", "A", "1", "200", "2026-01-01"), analysisTrade("2", "buy", "A", "1", "100", "2026-01-01"), analysisTrade("11", "sell", "A", "1", "300", "2026-01-01")}, "1", "300", "200", "200"},
		{"split fractional lots with finite total", []operations.Operation{analysisTrade("1", "buy", "A", "1", "1", "2026-01-01"), analysisTrade("2", "buy", "A", "2", "4", "2026-01-02"), analysisSplit("3", "3", "1", "2026-01-03"), analysisTrade("4", "sell", "A", "0.5", "10", "2026-01-04")}, "0.5", "10", "6", "3"},
		{"past oversell stays unavailable after purchase", []operations.Operation{analysisTrade("1", "sell", "A", "1", "100", "2026-01-01"), analysisTrade("2", "buy", "A", "2", "300", "2026-01-02")}, "1", "150", "", ""},
		{"partial sale", []operations.Operation{analysisTrade("1", "buy", "A", "10", "1000", "2026-01-01"), analysisTrade("2", "sell", "A", "4", "600", "2026-01-02")}, "6", "900", "100", "600"},
		{"sale proceeds do not change remaining cost", []operations.Operation{analysisTrade("1", "buy", "A", "2", "100", "2026-01-01"), analysisTrade("2", "sell", "A", "1", "100", "2026-01-02")}, "1", "100", "50", "50"},
		{"sale price does not affect average", []operations.Operation{analysisTrade("1", "buy", "A", "2", "1", "2026-01-01"), analysisTrade("2", "sell", "A", "1", "2.005", "2026-01-02")}, "1", "2.01", "0.5", "0.5"},
		{"sale without purchases", []operations.Operation{analysisTrade("1", "sell", "A", "2", "100", "2026-01-01")}, "-2", "-100", "", ""},
		{"split after partial sale", []operations.Operation{analysisTrade("1", "buy", "A", "10", "1000", "2026-01-01"), analysisTrade("2", "sell", "A", "4", "600", "2026-01-02"), analysisSplit("3", "1", "2", "2026-01-03")}, "12", "900", "50", "600"},
		{"fractions and expenses", []operations.Operation{analysisTrade("1", "buy", "A", "0.1", "0.01", "2026-01-01"), analysisTrade("2", "buy", "A", "0.2", "0.04", "2026-01-02")}, "0.3", "0.06", "0.17", "0.05"},
		{"split then buy same day", []operations.Operation{analysisTrade("1", "buy", "A", "2", "200", "2026-01-01"), analysisTrade("2", "buy", "A", "2", "60", "2026-01-02"), analysisSplit("3", "1", "2", "2026-01-02")}, "6", "180", "43.33", "260"},
		{"reverse split periodic price", []operations.Operation{analysisTrade("1", "buy", "A", "3", "1", "2026-01-01"), analysisSplit("2", "3", "1", "2026-01-02")}, "1", "1", "1", "1"},
		{"new entry uses new purchase cost", []operations.Operation{analysisTrade("1", "buy", "A", "1", "100", "2026-01-01"), analysisTrade("2", "sell", "A", "1", "110", "2026-01-02"), analysisTrade("3", "buy", "A", "1", "200", "2026-01-03")}, "1", "200", "200", "200"},
		{"future excluded", []operations.Operation{analysisTrade("1", "buy", "A", "2", "200", "2026-01-01"), analysisTrade("2", "buy", "A", "1", "500", "2027-01-01")}, "2", "200", "100", "200"},
		{"numeric ID latest buy", []operations.Operation{analysisTrade("10", "buy", "A", "1", "120", "2026-01-01"), analysisTrade("2", "buy", "A", "1", "100", "2026-01-01")}, "2", "240", "110", "220"},
		{"half up", []operations.Operation{analysisTrade("1", "buy", "A", "1", "1.005", "2026-01-01")}, "1", "1.01", "1.01", "1.01"},
		{"negative quantity", []operations.Operation{analysisTrade("1", "buy", "A", "1", "1.005", "2026-01-01"), analysisTrade("2", "sell", "A", "2", "5", "2026-01-02")}, "-1", "-2.5", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, currency := range []string{"RUB", "USD"} {
				out, err := BuildAnalysis(AnalysisSnapshot{Currency: currency, Operations: tt.ops}, "1", "2026-10-14")
				if err != nil || len(out.Assets) != 1 {
					t.Fatal(out, err)
				}
				a := out.Assets[0]
				if a.Quantity != tt.quantity || a.CurrentValue != tt.current || a.AveragePrice != tt.avg || a.AverageValue != tt.value || out.Currency != currency {
					t.Fatal(a, out)
				}
			}
		})
	}
}
func TestAnalysisDistributionAndMissingPrices(t *testing.T) {
	ops := []operations.Operation{analysisTrade("1", "buy", "A", "1", "1.004", "2026-01-01"), analysisTrade("2", "buy", "B", "1", "2.004", "2026-01-01")}
	out, err := BuildAnalysis(AnalysisSnapshot{Currency: "RUB", Operations: ops}, "1", "2026-01-01")
	if err != nil || out.Total != "3.01" || out.Assets[0].Percent != "33.38" || out.Assets[1].Percent != "66.62" {
		t.Fatal(out, err)
	}
	ops = append(ops, analysisTrade("3", "sell", "A", "1", "999", "2026-01-02"))
	out, err = BuildAnalysis(AnalysisSnapshot{Operations: ops}, "1", "2026-01-02")
	if err != nil || len(out.Assets) != 1 || out.Assets[0].Ticker != "B" || out.Assets[0].Percent != "100" {
		t.Fatal(out, err)
	}
	ops = append(ops, analysisTrade("4", "sell", "C", "1", "999", "2026-01-02"))
	out, err = BuildAnalysis(AnalysisSnapshot{Operations: ops}, "1", "2026-01-02")
	if err != nil || !out.Complete || !out.Negative || out.Total != "-997" || len(out.Issues) != 1 || out.Assets[0].Percent != "" {
		t.Fatal(out, err)
	}
	empty, err := BuildAnalysis(AnalysisSnapshot{}, "1", "2026-01-01")
	if err != nil || empty.Total != "0" || len(empty.Assets) != 0 {
		t.Fatal(empty, err)
	}
}
func TestAnalysisRejectsInvalidHistory(t *testing.T) {
	for _, ops := range [][]operations.Operation{
		{analysisTrade("1", "buy", "A", "1", "1", "2026-01-01"), analysisSplit("2", "3", "1", "2026-01-02")},
		{analysisTrade("1", "buy", "A", "0", "1", "2026-01-01")},
		{{ID: "1", PortfolioID: "2", Input: operations.Input{Date: "2026-01-01"}}},
	} {
		if _, err := BuildAnalysis(AnalysisSnapshot{Operations: ops}, "1", "2026-01-03"); err == nil {
			t.Fatal("accepted invalid history")
		}
	}
}

func TestAnalysisChartWeightsRemainBounded(t *testing.T) {
	huge := "1" + strings.Repeat("0", 400)
	out, err := BuildAnalysis(AnalysisSnapshot{Operations: []operations.Operation{
		analysisTrade("1", "buy", "A", "1", huge, "2026-01-01"),
		analysisTrade("2", "buy", "B", "1", huge, "2026-01-01"),
	}}, "1", "2026-01-01")
	if err != nil || out.Assets[0].ChartValue != "50" || out.Assets[1].ChartValue != "50" {
		t.Fatal(out, err)
	}
	tiny := "0." + strings.Repeat("0", 30) + "1"
	out, err = BuildAnalysis(AnalysisSnapshot{Operations: []operations.Operation{
		analysisTrade("1", "buy", "A", "1", "1", "2026-01-01"),
		analysisTrade("2", "buy", "B", "1", tiny, "2026-01-01"),
	}}, "1", "2026-01-01")
	// The saved trade price is zero after its existing 18-place rounding rule.
	if err != nil || out.Assets[1].CurrentValue != "0" || out.Assets[1].ChartValue != "0" {
		t.Fatal(out, err)
	}
	out, err = BuildAnalysis(AnalysisSnapshot{Operations: []operations.Operation{
		analysisTrade("1", "buy", "A", "1", huge, "2026-01-01"),
		analysisTrade("2", "buy", "B", "1", "1", "2026-01-01"),
	}}, "1", "2026-01-01")
	if err != nil || out.Assets[1].ChartValue != "" || out.Assets[1].CurrentValue != "1" {
		t.Fatal(out, err)
	}
}

func TestDividendsDoNotChangePurchaseAnalysis(t *testing.T) {
	s := AnalysisSnapshot{Currency: "RUB", Operations: []operations.Operation{analysisTrade("1", "buy", "A", "2", "200", "2026-01-01")}}
	before, err := BuildAnalysis(s, "1", "2026-01-03")
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range []string{"", "A", "B"} {
		s.Operations = append(s.Operations, operations.Operation{ID: "2", PortfolioID: "1", Input: operations.Input{Kind: "dividend", Date: "2026-01-02", AssetID: asset, Amount: "10", PortfolioAmount: "10"}})
	}
	after, err := BuildAnalysis(s, "1", "2026-01-03")
	if err != nil || after.Total != before.Total || len(after.Assets) != 1 || after.Assets[0] != before.Assets[0] {
		t.Fatal(before, after, err)
	}
}

func TestAnalysisClosedPositionAverageIsZero(t *testing.T) {
	ops := []operations.Operation{analysisTrade("1", "buy", "A", "1", "100", "2026-01-01"), analysisTrade("2", "sell", "A", "1", "150", "2026-01-02")}
	positions, _, err := prepareAnalysis(AnalysisSnapshot{Operations: ops}, "1", "2026-01-02", false)
	if err != nil {
		t.Fatal(err)
	}
	p := positions["A"]
	if p.averagePrice().Sign() != 0 || len(p.lots) != 0 {
		t.Fatal("closed position must have zero average and no remaining lots")
	}
	out, err := BuildAnalysis(AnalysisSnapshot{Operations: ops}, "1", "2026-01-02")
	if err != nil || len(out.Assets) != 0 || out.Total != "0" {
		t.Fatal(out, err)
	}
}
