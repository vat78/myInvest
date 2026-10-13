package cashflow

import "testing"

func TestAmounts(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{" 000,1000 ", "0.1"}, {"123456789012345678901234567890.000000000000000000001", "123456789012345678901234567890.000000000000000000001"}, {"1.2345678901234567890123456789", "1.2345678901234567890123456789"}, {"0010.000", "10"}} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Amount(tc.in)
			if err != nil || got != tc.want {
				t.Fatalf("%q %v", got, err)
			}
		})
	}
	for _, in := range []string{"", "0", "0,000", "-1", "+1", "1e2", "1 000", "1,000.00", ".1", "1.", "NaN", "1/2"} {
		if _, err := Amount(in); err == nil {
			t.Errorf("accepted %q", in)
		}
	}
}
func TestRates(t *testing.T) {
	for _, tc := range []struct{ a, b, want string }{{"1000", "94500", "94.5"}, {"500", "48000", "96"}, {"3", "1", "0.3333"}, {"6", "1", "0.1667"}, {"20000", "1", "0.0001"}, {"20000", "199999", "10"}, {"20001", "1", "0"}, {"0.000000000000000000001", "100000000000000000000", "100000000000000000000000000000000000000000"}, {"0,1", "0.1", "1"}} {
		t.Run(tc.a+"/"+tc.b, func(t *testing.T) {
			got, err := Rate(tc.a, tc.b)
			if err != nil || got != tc.want {
				t.Fatalf("got %s err %v want %s", got, err, tc.want)
			}
		})
	}
}
func TestBalance(t *testing.T) {
	for _, tc := range []struct {
		amounts, kinds []string
		want           string
	}{
		{nil, nil, "0"},
		{[]string{"0.1", "0.2"}, []string{"deposit", "deposit"}, "0.3"},
		{[]string{"94500", "48000"}, []string{"deposit", "withdrawal"}, "46500"},
		{[]string{"0.123456789012345678901", "0.1234567890123456789"}, []string{"deposit", "withdrawal"}, "0.000000000000000000001"},
		{[]string{"0.1", "0.2"}, []string{"deposit", "withdrawal"}, "-0.1"},
		{[]string{"1", "1"}, []string{"deposit", "withdrawal"}, "0"},
	} {
		ops := []Operation{}
		for i, a := range tc.amounts {
			ops = append(ops, Operation{Input: Input{Kind: tc.kinds[i], Amount: "1000", PortfolioAmount: a, CurrencyCode: "USD"}})
		}
		if got := Balance(ops); got != tc.want {
			t.Errorf("got %s want %s", got, tc.want)
		}
	}
}
func TestValidation(t *testing.T) {
	base := Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1.0", PortfolioAmount: "1", Comment: "комментарий"}
	if got, err := Validate(base, "RUB"); err != nil || got.Amount != "1" || got.Comment != base.Comment {
		t.Fatal(got, err)
	}
	for _, tc := range []struct{ field, value string }{{"kind", "trade"}, {"date", "2026-02-30"}, {"date", "0000-01-01"}, {"date", "2026-1-01"}, {"currencyCode", ""}, {"amount", "0"}, {"portfolioAmount", "-1"}, {"portfolioAmount", "2"}} {
		in := base
		switch tc.field {
		case "kind":
			in.Kind = tc.value
		case "date":
			in.Date = tc.value
		case "currencyCode":
			in.CurrencyCode = tc.value
		case "amount":
			in.Amount = tc.value
		case "portfolioAmount":
			in.PortfolioAmount = tc.value
		}
		_, err := Validate(in, "RUB")
		f, ok := err.(*FieldError)
		if !ok || f.Field != tc.field {
			t.Errorf("%v: %v", tc, err)
		}
	}
	for _, date := range []string{"1900-01-01", "9999-12-31", "2024-02-29"} {
		in := base
		in.Date = date
		if _, err := Validate(in, "RUB"); err != nil {
			t.Fatal(err)
		}
	}
	base.CurrencyCode = "USD"
	base.PortfolioAmount = "94.5"
	if _, err := Validate(base, "RUB"); err != nil {
		t.Fatal(err)
	}
}
