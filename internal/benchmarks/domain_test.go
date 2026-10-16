package benchmarks

import "testing"

func TestExactCalculations(t *testing.T) {
	for _, tc := range []struct{ base, mode, input, value, growth string }{
		{"100", "growth", "10", "110", "10"}, {"110", "growth", "-5", "104.5", "-5"},
		{"0.123456789012345678901", "growth", "0.1", "0.123580245801358024579901", "0.1"},
		{"3", "value", "4", "4", "33.333333333333333333"},
		{"3", "value", "2", "2", "-33.333333333333333333"},
		{"100", "growth", "0", "100", "0"},
		{"100", "growth", "-99.99", "0.01", "-99.99"},
		{"1", "value", "1.000000000000000000005", "1.000000000000000000005", "0.000000000000000001"},
	} {
		t.Run(tc.base+tc.mode+tc.input, func(t *testing.T) {
			got, e := Recalculate([]Point{{BenchmarkID: "1", Date: "2026-01-01", Mode: "value", Input: tc.base}, {BenchmarkID: "1", Date: "2026-02-01", Mode: tc.mode, Input: tc.input}})
			if e != nil {
				t.Fatal(e)
			}
			if got[0].Growth != "" || got[1].Value != tc.value || got[1].Growth != tc.growth {
				t.Fatalf("%+v", got)
			}
		})
	}
}
func TestInvalidInputs(t *testing.T) {
	for _, tc := range []struct{ mode, input string }{{"value", "0"}, {"value", "-1"}, {"growth", "-100"}, {"growth", "-100.01"}, {"growth", "1e2"}, {"growth", "1/2"}, {"growth", "+1"}, {"growth", "-.1"}, {"bad", "1"}} {
		if _, e := ValidatePoint(Point{BenchmarkID: "1", Date: "2026-01-01", Mode: tc.mode, Input: tc.input}); e == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	for _, p := range []Point{{BenchmarkID: "01", Date: "2026-01-01", Mode: "value", Input: "1"}, {BenchmarkID: "1", Date: "2026-02-29", Mode: "value", Input: "1"}, {BenchmarkID: "1", Date: "0000-01-01", Mode: "value", Input: "1"}} {
		if _, e := ValidatePoint(p); e == nil {
			t.Fatal(p)
		}
	}
	if _, e := Recalculate([]Point{{BenchmarkID: "1", Date: "2026-01-01", Mode: "growth", Input: "0"}}); e == nil {
		t.Fatal("first percent accepted")
	}
	p, e := ValidatePoint(Point{BenchmarkID: "1", Date: "2026-01-01", Mode: "value", Input: "00012,123456789012345678900"})
	if e != nil || p.Input != "12.123456789012345678900" {
		t.Fatal(p, e)
	}
}

func TestSources(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		invalid     bool
	}{
		{"", "", false},
		{"  https://example.org/stats?q=1&year=2026 \r\n\n http://example.com/данные ", "https://example.org/stats?q=1&year=2026\nhttp://example.com/данные", false},
		{"javascript:alert(1)", "", true}, {"/statistics", "", true},
		{"https://", "", true}, {"https://user:password@example.com", "", true},
		{"https://example.com\nfile:///tmp/data", "", true},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ValidateSources(tc.input)
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}
