package currency

import "testing"

func TestDecimal(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"95,125", "95.125"}, {" 00095.12500 ", "95.12500"}, {"000.00001", "0.00001"},
		{"1", "1"}, {"0.123456789012345678901234567890123456789", "0.123456789012345678901234567890123456789"},
		{"999999999999999999999999999999.123456789", "999999999999999999999999999999.123456789"},
		{"", ""}, {"0", ""}, {"000,000", ""}, {"-1", ""}, {"+1", ""}, {"NaN", ""},
		{"1e3", ""}, {"1,2.3", ""}, {"1..2", ""}, {".5", ""}, {"5.", ""}, {"1 000", ""}, {"１２", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Decimal(tc.input)
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
func TestCode(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{" usd ", "USD"}, {"rUb", "RUB"}, {"XYZ", "XYZ"}, {"", ""}, {"US", ""}, {"USDD", ""}, {"R1B", ""}, {"РУБ", ""}, {"U SD", ""}, {"ſud", ""},
	} {
		got, err := Code(tc.input)
		if got != tc.want || (err != nil) != (tc.want == "") {
			t.Fatalf("%q: %q %v", tc.input, got, err)
		}
	}
}
func TestPair(t *testing.T) {
	for _, tc := range []struct {
		date, base, quote string
		valid             bool
	}{
		{"2024-02-29", " usd ", "rub", true}, {"2026-10-12", "EUR", "RUB", true},
		{"2026-02-29", "USD", "RUB", false}, {"2026-04-31", "USD", "RUB", false},
		{"2026-1-01", "USD", "RUB", false}, {"0000-01-01", "USD", "RUB", false},
		{"2026-10-12T00:00:00Z", "USD", "RUB", false}, {"", "USD", "RUB", false},
		{"2026-10-12", " usd", "USD", false}, {"2026-10-12", "123", "RUB", false},
	} {
		pair, err := NewPair(tc.date, tc.base, tc.quote)
		if (err == nil) != tc.valid {
			t.Fatalf("%+v: %+v %v", tc, pair, err)
		}
	}
}

func TestInverse(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"1", "1"}, {"2", "0.5"}, {"0.5", "2"}, {"0.00000000000000000001", "100000000000000000000"},
		{"95,125", "0.010512483574244415"}, {"95", "0.010526315789473684"},
		{"3", "0.333333333333333333"}, {"6", "0.166666666666666667"},
		{" 0002.000 ", "0.5"}, {"0.99999999999999999999", "1"},
		{"95.123456789012345678901234567890123456789000", "0.010512654120823639"},
		{"2000000000000000000", "0.000000000000000001"},
		{"1999999999999999999", "0.000000000000000001"},
		{"2000000000000000001", ""}, {"4000000000000000000", ""},
		{"99999999999999999999999999999999", ""},
		{"0", ""}, {"-1", ""}, {"", ""}, {"1e3", ""}, {"NaN", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Inverse(tc.input)
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
