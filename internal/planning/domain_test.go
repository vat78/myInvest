package planning

import (
	"math/big"
	"testing"
)

func parameters() Parameters {
	return Parameters{CurrencyCode: "RUB", StartYear: "2026", Years: "2", Mode: "contribution", Goal: "0", Contribution: "100", ReturnPercent: "10", InflationPercent: "0", BenchmarkID: "1", InitialCapital: "0"}
}
func TestAnnualCalculations(t *testing.T) {
	for _, tc := range []struct{ name, n, mode, initial, a, goal, r, inflation, wantA, final, real, excess string }{
		{"one year", "1", "contribution", "0", "100", "0", "10", "0", "100", "110", "110", ""},
		{"beginning of year", "2", "contribution", "0", "100", "0", "10", "0", "100", "231", "231", ""},
		{"zero rates", "2", "goal", "10", "0", "100", "0", "0", "45", "100", "100", "0"},
		{"deflation", "2", "contribution", "0", "100", "0", "0", "-10", "100", "190", "234.567901234567901235", ""},
		{"goal ceil", "1", "goal", "0", "0", "100", "10", "0", "90.91", "100.001", "100.001", "0.001"},
		{"enough capital", "1", "goal", "100", "0", "100", "10", "0", "0", "110", "110", "10"},
		{"indexed goal", "2", "goal", "0", "0", "100", "0", "10", "57.62", "121.002", "100.001652892561983471", "0.001652892561983471"},
		{"fractions", "1", "contribution", "0.1234567890123456789", "0.0000000000000000001", "0", "0", "0", "0.0000000000000000001", "0.123456789012345679", "0.123456789012345679", ""},
		{"negative initial and return", "1", "goal", "-5", "0", "10", "-50", "0", "25", "10", "10", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := parameters()
			p.Years = tc.n
			p.Mode = tc.mode
			p.InitialCapital = tc.initial
			p.Contribution = tc.a
			p.Goal = tc.goal
			p.ReturnPercent = tc.r
			p.InflationPercent = tc.inflation
			f, e := Resolve(p, nil, "2026-10-14")
			if e != nil {
				t.Fatal(e)
			}
			out, e := Calculate(p, rat(tc.initial), f)
			if e != nil || !out.Complete || out.FirstContribution != tc.wantA || out.FinalCapital != tc.final || out.FinalRealCapital != tc.real || out.Excess != tc.excess {
				t.Fatalf("%+v %v", out, e)
			}
			if len(out.Rows) != len(f) || len(out.Points) != len(f)+1 {
				t.Fatal("wrong terminal point")
			}
			if tc.mode == "goal" && !out.GoalReached {
				t.Fatal("goal missed")
			}
		})
	}
}
func TestHistoricalBoundariesAndMissing(t *testing.T) {
	p := parameters()
	p.StartYear = "2024"
	p.Years = "4"
	points := []IndexPoint{{"2023-12-31", "3"}, {"2024-12-31", "4"}, {"2025-12-30", "9"}, {"2026-12-31", "999"}, {"2027-12-31", "9999"}}
	f, e := Resolve(p, points, "2026-10-14")
	if e != nil {
		t.Fatal(e)
	}
	if f[0].Value != "4/3" || f[0].Source != "historical" || f[1].Source != "missing" || f[2].Value != "1" || f[2].Source != "expected" || f[3].Source != "expected" {
		t.Fatal(f)
	}
	out, e := Calculate(p, new(big.Rat), f)
	if e != nil || out.Complete || len(out.Rows) != 1 || out.FinalCapital != "" {
		t.Fatal(out, e)
	}
	p.Mode = "goal"
	p.Goal = "100"
	out, e = Calculate(p, new(big.Rat), f)
	if e != nil || out.FirstContribution != "" || len(out.Rows) != 0 {
		t.Fatal(out, e)
	}
	p.AllowEstimated = true
	f, e = Resolve(p, points, "2026-10-14")
	if e != nil || f[1].Source != "estimated" {
		t.Fatal(f, e)
	}
	out, e = Calculate(p, new(big.Rat), f)
	if e != nil || !out.Complete {
		t.Fatal(out, e)
	}
}
func TestValidation(t *testing.T) {
	for _, edit := range []func(*Parameters){func(p *Parameters) { p.ReturnPercent = "-100" }, func(p *Parameters) { p.InflationPercent = "-100.1" }, func(p *Parameters) { p.Years = "0" }, func(p *Parameters) { p.Years = "1.5" }, func(p *Parameters) { p.StartYear = "1" }, func(p *Parameters) { p.StartYear = "9998"; p.Years = "2" }, func(p *Parameters) { p.Contribution = "-1" }, func(p *Parameters) { p.Contribution = "1/3" }, func(p *Parameters) { p.Contribution = "1e3" }, func(p *Parameters) { p.Mode = "unknown" }, func(p *Parameters) { p.BenchmarkID = "0" }, func(p *Parameters) { p.CurrencyCode = "bad!" }} {
		p := parameters()
		edit(&p)
		if _, e := Validate(p); e == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
}
