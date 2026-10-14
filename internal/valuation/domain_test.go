package valuation

import (
	"math/big"
	"testing"
)

func TestRounding(t *testing.T) {
	for _, tc := range []struct {
		input string
		scale int
		want  string
	}{{"1/3", 18, "0.333333333333333333"}, {"-1/6", 18, "-0.166666666666666667"}, {"1/200", 2, "0.01"}, {"-1/200", 2, "-0.01"}, {"-1/1000", 2, "0"}, {"123456789012345678901", 18, "123456789012345678901"}} {
		r, _ := new(big.Rat).SetString(tc.input)
		if got := Format(r, tc.scale); got != tc.want {
			t.Fatalf("%+v: %s", tc, got)
		}
	}
}
func TestOverviewHistoricalRatesAndFuture(t *testing.T) {
	s := Snapshot{Portfolios: []Portfolio{
		{ID: "1", Name: "RUB", Currency: "RUB", Days: []Day{{Date: "2026-01-01", Cash: "750", Positions: []Position{{Ticker: "A", TypeCode: "stocks", Quantity: "2.5", Numerator: "250", Denominator: "1"}}}, {Date: "2026-01-04", Cash: "9000"}}},
		{ID: "2", Name: "USD", Currency: "USD", Days: []Day{{Date: "2026-01-02", Cash: "100"}}}}, Rates: []Rate{{Date: "2026-01-02", Currency: "USD", Value: "95"}, {Date: "2026-01-03", Currency: "USD", Value: "96"}, {Date: "2026-01-04", Currency: "USD", Value: "200"}}}
	out, err := Build(s, "2026-01-03")
	if err != nil {
		t.Fatal(err)
	}
	if out.Total.Value != "10600" || len(out.Points) != 3 || out.Points[0].Value != "1000" || out.Points[1].Value != "10500" || out.Points[2].Value != "10600" || out.Portfolios[1].RateDate != "2026-01-03" {
		t.Fatal(out)
	}
	if out.Portfolios[0].Summary.Groups[0].Percent != "25" || out.Portfolios[0].Summary.Value != "1000" {
		t.Fatal(out.Portfolios)
	}
	s.Rates = nil
	out, err = Build(s, "2026-01-03")
	if err != nil || out.Total.Complete || out.Total.Value != "" || !out.Points[0].Complete || out.Points[1].Complete || !out.Portfolios[1].Summary.Complete {
		t.Fatal(out, err)
	}
}
func TestSpecialStates(t *testing.T) {
	for _, tc := range []struct {
		name, cash, q, n, d, code string
		complete, negative        bool
		value                     string
	}{{"empty", "0", "0", "", "", "", true, false, "0"}, {"unclassified", "0", "1", "1", "3", "", true, false, "0.333333333333333333"}, {"missing", "1", "1", "", "", "stocks", false, false, ""}, {"negative cash", "-1", "1", "1", "1", "stocks", true, true, "0"}, {"negative position", "2", "-1", "-1", "1", "stocks", true, true, "1"}, {"negative quantity at zero price", "0", "-1", "0", "1", "stocks", true, true, "0"}} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Build(Snapshot{Portfolios: []Portfolio{{Name: "P", Currency: "RUB", Days: []Day{{Date: "2026-01-01", Cash: tc.cash, Positions: []Position{{Ticker: "A", TypeCode: tc.code, Quantity: tc.q, Numerator: tc.n, Denominator: tc.d}}}}}}}, "2026-01-01")
			if err != nil || out.Total.Complete != tc.complete || out.Total.Negative != tc.negative || out.Total.Value != tc.value {
				t.Fatal(out, err)
			}
			if tc.negative {
				for _, g := range out.Total.Groups {
					if g.Percent != "" {
						t.Fatal(g)
					}
				}
			}
		})
	}
	out, err := Build(Snapshot{}, "2026-01-01")
	if err != nil || len(out.Points) != 0 || out.Total.Value != "0" || out.Portfolios == nil {
		t.Fatal(out, err)
	}
	for _, d := range []string{"0000-01-01", "2026-02-30", "2026-1-1"} {
		if _, err := Build(Snapshot{}, d); err == nil {
			t.Fatal(d)
		}
	}
}
func TestMalformedStorage(t *testing.T) {
	for _, pos := range []Position{{Quantity: "bad"}, {Quantity: "1", Numerator: "1", Denominator: "0"}, {Quantity: "1", Numerator: "1", Denominator: "1", TypeCode: "bad"}} {
		if _, err := Build(Snapshot{Portfolios: []Portfolio{{Currency: "RUB", Days: []Day{{Date: "2026-01-01", Cash: "0", Positions: []Position{pos}}}}}}, "2026-01-01"); err == nil {
			t.Fatal(pos)
		}
	}
}

func TestCumulativeDepositsIndependentOfValuation(t *testing.T) {
	s := Snapshot{Portfolios: []Portfolio{
		{Name: "R", Currency: "RUB", Days: []Day{{Date: "2026-01-01", Cash: "0.1", Deposits: "0.1"}, {Date: "2026-01-03", Cash: "-1", Deposits: "0.2"}, {Date: "2026-01-05", Cash: "100", Deposits: "100"}}},
		{Name: "U", Currency: "USD", Days: []Day{{Date: "2026-01-02", Cash: "10", Deposits: "900.1234567890123456789"}}},
	}, Rates: []Rate{{Date: "2026-01-02", Currency: "USD", Value: "95"}, {Date: "2026-01-03", Currency: "USD", Value: "200"}}}
	for _, rates := range [][]Rate{s.Rates, nil} {
		s.Rates = rates
		out, err := Build(s, "2026-01-04")
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range []string{"0.1", "900.2234567890123456789", "900.3234567890123456789", "900.3234567890123456789"} {
			if out.Points[i].Deposits != want || !out.Points[i].DepositsComplete {
				t.Fatal(out.Points[i])
			}
		}
	}
	s.Portfolios[1].Days[0].Deposits = ""
	out, err := Build(s, "2026-01-03")
	if err != nil || !out.Points[0].DepositsComplete || out.Points[1].DepositsComplete || out.Points[1].Deposits != "" || len(out.Points[1].DepositIssues) != 1 {
		t.Fatal(out, err)
	}
}
