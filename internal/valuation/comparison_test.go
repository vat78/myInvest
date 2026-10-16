package valuation

import (
	"myinvest/internal/benchmarks"
	"myinvest/internal/quotes"
	"testing"
)

func comparisonFixture() ComparisonSnapshot {
	return ComparisonSnapshot{Benchmarks: []ComparisonBenchmark{{Benchmark: benchmarks.Benchmark{ID: "1", Name: "Inflation"}, Points: []benchmarks.Point{{Date: "2023-12-31", Value: "100"}, {Date: "2024-12-31", Value: "120"}, {Date: "2025-12-31", Value: "132"}}}}}
}
func TestReturnBenchmarkFXAndNegative(t *testing.T) {
	for _, tc := range []struct{ currency, firstFX, lastFX, last, annual, cumulative string }{
		{"RUB", "", "", "90", "-10", "-10"},
		{"USD", "90", "99", "110", "21", "21"},
		{"", "", "", "100", "0", "0"},
		{"RUB", "", "", "100.000000000000000001", "0.000000000000000001", "0.000000000000000001"},
	} {
		t.Run(tc.currency+tc.last, func(t *testing.T) {
			s := comparisonFixture()
			s.Benchmarks = append(s.Benchmarks, ComparisonBenchmark{Benchmark: benchmarks.Benchmark{ID: "2", Name: "Other", CurrencyCode: tc.currency}, Points: []benchmarks.Point{{Date: "2023-12-31", Value: "100"}, {Date: "2024-12-31", Value: tc.last}}})
			if tc.firstFX != "" {
				s.Rates = []Rate{{Date: "2023-12-31", Currency: "USD", Value: tc.firstFX}, {Date: "2024-12-31", Currency: "USD", Value: tc.lastFX}}
			}
			out, e := BuildReturnComparison(s, "1", "2024-12-31")
			if e != nil {
				t.Fatal(e)
			}
			p := out.Series[1].Points[1]
			if p.Annual != tc.annual || p.Cumulative != tc.cumulative {
				t.Fatal(p)
			}
		})
	}
}
func TestReturnPortfolioFlowsAndInflation(t *testing.T) {
	for _, tc := range []struct{ name, currency, first, second, flow, kind, annual, cumulative string }{
		{"gain", "RUB", "100", "110", "", "", "10", "32"},
		{"deposit", "RUB", "100", "210", "100", "deposit", "10", "32"},
		{"withdrawal", "RUB", "100", "60", "50", "withdrawal", "10", "32"},
		{"loss", "RUB", "100", "90", "", "", "-10", "8"},
		{"fees", "RUB", "99", "99", "", "", "-1", "18.8"},
		{"dividend", "RUB", "100", "105", "", "", "5", "26"},
		{"fx", "USD", "100", "100", "", "", "10", "32"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := comparisonFixture()
			s.Portfolios = []Portfolio{{ID: "1", Name: "Portfolio", Currency: tc.currency, Days: []Day{{Date: "2025-01-01", Cash: tc.first}, {Date: "2025-12-31", Cash: tc.second}}}}
			s.Flows = []ComparisonFlow{{PortfolioID: "1", Date: "2025-01-01", Kind: "deposit", Amount: "100"}}
			if tc.flow != "" {
				s.Flows = append(s.Flows, ComparisonFlow{PortfolioID: "1", Date: "2025-12-31", Kind: tc.kind, Amount: tc.flow})
			}
			s.Rates = []Rate{{Date: "2025-01-01", Currency: "USD", Value: "90"}, {Date: "2025-12-31", Currency: "USD", Value: "99"}}
			out, e := BuildReturnComparison(s, "1", "2025-12-31")
			if e != nil {
				t.Fatal(e)
			}
			p := out.Series[1].Points[2]
			if p.Annual != tc.annual || p.Cumulative != tc.cumulative {
				t.Fatal(p)
			}
			if out.Series[1].Points[1].Cumulative != "" {
				t.Fatal("portfolio before formation")
			}
		})
	}
}
func TestReturnLinkedPeriodsAndReinvestment(t *testing.T) {
	s := comparisonFixture()
	s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{{Date: "2023-12-31", Cash: "100"}, {Date: "2024-06-01", Cash: "210"}, {Date: "2024-12-31", Cash: "231"}, {Date: "2025-01-01", Cash: "0"}, {Date: "2025-02-01", Cash: "100"}, {Date: "2025-12-31", Cash: "110"}}}}
	s.Flows = []ComparisonFlow{{"1", "2023-12-31", "deposit", "100"}, {"1", "2024-06-01", "deposit", "100"}, {"1", "2025-01-01", "withdrawal", "231"}, {"1", "2025-02-01", "deposit", "100"}}
	out, e := BuildReturnComparison(s, "1", "2025-12-31")
	if e != nil {
		t.Fatal(e)
	}
	ps := out.Series[1].Points
	if ps[1].Annual != "21" || ps[1].Cumulative != "21" || ps[2].Annual != "10" || ps[2].Cumulative != "33.1" {
		t.Fatal(ps)
	}
}
func TestReturnMissingDataAndNonpositive(t *testing.T) {
	for _, tc := range []struct {
		name, currency, cash, flow string
		rates                      []Rate
	}{
		{name: "no FX", currency: "USD", cash: "100", flow: "100"},
		{name: "zero", currency: "RUB", cash: "0", flow: "0"},
		{name: "negative", currency: "RUB", cash: "-1", flow: "100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := comparisonFixture()
			s.Portfolios = []Portfolio{{ID: "1", Currency: tc.currency, Days: []Day{{Date: "2024-01-01", Cash: tc.cash}}}}
			s.Flows = []ComparisonFlow{{"1", "2024-01-01", "deposit", tc.flow}}
			s.Rates = tc.rates
			out, e := BuildReturnComparison(s, "1", "2025-12-31")
			if e != nil {
				t.Fatal(e)
			}
			for _, p := range out.Series[1].Points[1:] {
				if p.Issue == "" || p.Cumulative != "" || p.Annual != "" {
					t.Fatal(p)
				}
			}
		})
	}
	s := comparisonFixture()
	if _, e := BuildReturnComparison(s, "999", "2025-12-31"); e == nil {
		t.Fatal("unknown inflation")
	}
	s.Benchmarks[0].Benchmark.CurrencyCode = "USD"
	if _, e := BuildReturnComparison(s, "1", "2025-12-31"); e == nil {
		t.Fatal("foreign inflation")
	}
}
func TestReturnExistingPortfolioAndPartialYear(t *testing.T) {
	s := comparisonFixture()
	s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{{Date: "2023-12-01", Cash: "100"}, {Date: "2024-07-01", Cash: "110"}, {Date: "2025-01-01", Cash: "999"}}}}
	s.Flows = []ComparisonFlow{{"1", "2023-12-01", "deposit", "100"}, {"1", "2025-01-01", "deposit", "889"}}
	out, e := BuildReturnComparison(s, "1", "2024-07-01")
	if e != nil {
		t.Fatal(e)
	}
	if out.Dates[1] != "2024-07-01" || out.Series[1].Points[1].Annual != "10" {
		t.Fatal(out)
	}
	// Quoted appreciation and commissions use exact prepared valuations.
	s.Portfolios[0].Days = []Day{{Date: "2024-01-01", Cash: "-1", Positions: []Position{{AssetID: "1", Quantity: "1", Numerator: "100", Denominator: "1", TradeDate: "2024-01-01"}}}}
	s.Flows = []ComparisonFlow{{"1", "2024-01-01", "deposit", "100"}}
	s.Pricing.Quotes = []quotes.Quote{{AssetID: "1", Date: "2024-07-01", CurrencyCode: "RUB", Price: "111"}}
	out, e = BuildReturnComparison(s, "1", "2024-07-01")
	if e != nil {
		t.Fatal(e)
	}
	if out.Series[1].Points[1].Annual != "10" {
		t.Fatal(out.Series[1])
	}
}

func TestReturnLateBenchmarkAndMissingFX(t *testing.T) {
	s := comparisonFixture()
	s.Benchmarks = append(s.Benchmarks, ComparisonBenchmark{Benchmark: benchmarks.Benchmark{ID: "2", Name: "Late", CurrencyCode: "USD"}, Points: []benchmarks.Point{{Date: "2025-01-01", Value: "100"}, {Date: "2025-12-31", Value: "110"}}})
	s.Rates = []Rate{{Date: "2025-01-01", Currency: "USD", Value: "90"}, {Date: "2025-12-31", Currency: "USD", Value: "99"}}
	out, e := BuildReturnComparison(s, "1", "2025-12-31")
	if e != nil {
		t.Fatal(e)
	}
	p := out.Series[1].Points[2]
	if p.Annual != "21" || p.Cumulative != "45.2" {
		t.Fatal(p)
	}
	s.Rates = nil
	out, e = BuildReturnComparison(s, "1", "2025-12-31")
	if e != nil {
		t.Fatal(e)
	}
	p = out.Series[1].Points[2]
	if p.Cumulative != "" || p.Issue == "" {
		t.Fatal(p)
	}
	s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{{Date: "2023-01-01", Cash: "-1"}}}}
	out, e = BuildReturnComparison(s, "1", "2025-12-31")
	if e != nil {
		t.Fatal(e)
	}
	if out.Series[2].Points[0].Cumulative != "" {
		t.Fatal(out.Series[2])
	}
}

func TestReturnIntermediateValuationIssuesRemainUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*ComparisonSnapshot)
	}{
		{"quote", func(s *ComparisonSnapshot) {
			s.Pricing.Quotes = append(s.Pricing.Quotes, quotes.Quote{AssetID: "1", Date: "2024-05-01", CurrencyCode: "RUB", Price: "40"})
		}},
		{"snapshot", func(s *ComparisonSnapshot) {
			s.Portfolios[0].Days = append(s.Portfolios[0].Days,
				Day{Date: "2024-05-01", Cash: "-20"}, Day{Date: "2024-06-01", Cash: "120"})
		}},
		{"quote FX", func(s *ComparisonSnapshot) {
			s.Pricing.Quotes = []quotes.Quote{{AssetID: "1", Date: "2024-01-02", CurrencyCode: "USD", Price: "100"}}
			s.Pricing.FX = []ExchangeRate{{Date: "2024-01-02", Base: "USD", Quote: "RUB", Value: "1"},
				{Date: "2024-05-01", Base: "USD", Quote: "RUB", Value: "0.4"},
				{Date: "2024-12-31", Base: "USD", Quote: "RUB", Value: "1.1"}}
		}},
		{"missing FX", func(s *ComparisonSnapshot) {
			s.Pricing.Quotes = append(s.Pricing.Quotes, quotes.Quote{AssetID: "1", Date: "2024-05-01", CurrencyCode: "USD", Price: "100"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := comparisonFixture()
			s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{
				{Date: "2024-01-01", Cash: "100"},
				{Date: "2024-01-02", Cash: "-100", Positions: []Position{{AssetID: "1", Quantity: "2", Numerator: "200", Denominator: "1", TradeDate: "2024-01-02"}}},
			}}}
			s.Flows = []ComparisonFlow{{"1", "2024-01-01", "deposit", "100"}}
			s.Pricing.Quotes = []quotes.Quote{{AssetID: "1", Date: "2024-12-31", CurrencyCode: "RUB", Price: "110"}}
			tc.change(&s)
			out, err := BuildReturnComparison(s, "1", "2025-12-31")
			if err != nil {
				t.Fatal(err)
			}
			for _, point := range out.Series[1].Points[1:] {
				if point.Issue == "" || point.Annual != "" || point.Cumulative != "" {
					t.Fatal(point)
				}
			}
		})
	}
}

func TestReturnEmptySnapshotsBeforeFunding(t *testing.T) {
	for _, first := range []string{"2023-12-01", "2024-01-01"} {
		for _, tc := range []struct{ cash, finalCash, annual, cumulative string }{
			{"100", "110", "10", "32"}, {"99", "109", "9", "30.8"},
		} {
			t.Run(first+"/"+tc.cash, func(t *testing.T) {
				s := comparisonFixture()
				s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{
					{Date: first, Cash: "0", Positions: []Position{{AssetID: "1", Quantity: "0"}}},
					{Date: "2025-01-01", Cash: tc.cash}, {Date: "2025-12-31", Cash: tc.finalCash},
				}}}
				s.Flows = []ComparisonFlow{{"1", "2025-01-01", "deposit", "100"}}
				out, err := BuildReturnComparison(s, "1", "2025-12-31")
				if err != nil {
					t.Fatal(err)
				}
				series := out.Series[1]
				if series.StartDate != "2025-01-01" {
					t.Fatal(series)
				}
				for _, point := range series.Points[:2] {
					if point.Annual != "" || point.Cumulative != "" {
						t.Fatal(point)
					}
				}
				point := series.Points[2]
				if point.Issue != "" || point.Annual != tc.annual || point.Cumulative != tc.cumulative {
					t.Fatal(point)
				}
			})
		}
	}
}

func TestReturnIntermediatePositiveValuations(t *testing.T) {
	s := comparisonFixture()
	s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{
		{Date: "2024-01-01", Cash: "100"},
		{Date: "2024-01-02", Cash: "-100", Positions: []Position{{AssetID: "1", Quantity: "2", Numerator: "200", Denominator: "1", TradeDate: "2024-01-02"}}},
	}}}
	s.Flows = []ComparisonFlow{{"1", "2024-01-01", "deposit", "100"}}
	s.Pricing.Quotes = []quotes.Quote{
		{AssetID: "1", Date: "2024-05-01", CurrencyCode: "RUB", Price: "60"},
		{AssetID: "1", Date: "2024-12-31", CurrencyCode: "RUB", Price: "110"},
		{AssetID: "1", Date: "2025-01-01", CurrencyCode: "RUB", Price: "40"},
	}
	out, err := BuildReturnComparison(s, "1", "2024-12-31")
	if err != nil {
		t.Fatal(err)
	}
	point := out.Series[1].Points[1]
	if point.Issue != "" || point.Annual != "20" || point.Cumulative != "20" {
		t.Fatal(point)
	}
}

func TestReturnFirstFundingOnCommonBase(t *testing.T) {
	s := comparisonFixture()
	s.Portfolios = []Portfolio{{ID: "1", Currency: "RUB", Days: []Day{
		{Date: "2023-12-01", Cash: "0"}, {Date: "2023-12-31", Cash: "99"},
	}}}
	s.Flows = []ComparisonFlow{{"1", "2023-12-31", "deposit", "100"}}
	out, err := BuildReturnComparison(s, "1", "2023-12-31")
	if err != nil {
		t.Fatal(err)
	}
	point := out.Series[1].Points[0]
	if point.Issue != "" || point.Annual != "-1" || point.Cumulative != "-1" {
		t.Fatal(point)
	}
	s.Portfolios[0].Days = s.Portfolios[0].Days[:1]
	s.Flows = nil
	out, err = BuildReturnComparison(s, "1", "2024-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if out.Series[1].StartDate != "" || len(out.Series[1].Points) != 0 {
		t.Fatal(out.Series[1])
	}
}
