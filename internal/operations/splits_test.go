package operations

import (
	"errors"
	"strings"
	"testing"
)

func TestSplitQuantity(t *testing.T) {
	for _, tt := range []struct{ q, old, next, want string }{
		{"12", "1", "10", "120"}, {"12", "10", "1", "1.2"}, {"0.1", "1", "10", "1"},
		{"3", "3", "1", "1"}, {"0", "3", "1", "0"}, {"-12", "10", "1", "-1.2"},
		{"1", "1048576", "1", "0.00000095367431640625"},
		{"0.000000000000000000001", "10", "1", "0.0000000000000000000001"},
		{"999999999999999999999999999999", "1", "1000000000000000000000000", "999999999999999999999999999999000000000000000000000000"},
		{"1", "3", "1", ""}, {"-1", "3", "1", ""}, {"1", "6", "2", ""},
	} {
		t.Run(tt.q+":"+tt.old+":"+tt.next, func(t *testing.T) {
			got, err := SplitQuantity(tt.q, tt.old, tt.next)
			if tt.want == "" {
				if err == nil {
					t.Fatal("periodic result accepted", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatal(got, err, "want", tt.want)
			}
		})
	}
}
func TestSplitValidation(t *testing.T) {
	valid := Input{Kind: "split", Date: "2026-10-13", AssetID: "1", OldUnits: "0010", NewUnits: "01", Amount: "999", Quantity: "999", Comment: "keep"}
	got, err := ValidateSplit(valid)
	if err != nil || got.OldUnits != "10" || got.NewUnits != "1" || got.Amount != "" || got.Quantity != "" || got.Comment != "keep" {
		t.Fatal(got, err)
	}
	for _, field := range []string{"oldUnits", "newUnits"} {
		for _, value := range []string{"", "0", "-1", "1.5", "1,5", "1e2", "1/2", "+1"} {
			in := valid
			if field == "oldUnits" {
				in.OldUnits = value
			} else {
				in.NewUnits = value
			}
			_, err := ValidateSplit(in)
			var f *FieldError
			if !errors.As(err, &f) || f.Field != field {
				t.Fatal(field, value, err)
			}
		}
	}
	for _, change := range []func(*Input){func(i *Input) { i.NewUnits = "10" }, func(i *Input) { i.Date = "2026-02-30" }, func(i *Input) { i.Date = "0000-01-01" }, func(i *Input) { i.AssetID = "" }, func(i *Input) { i.Kind = "buy" }} {
		in := valid
		change(&in)
		if _, err := ValidateSplit(in); err == nil {
			t.Fatal(in)
		}
	}
}
func TestSplitHistoryOrderAndIntermediatePrecision(t *testing.T) {
	ops := []Operation{
		{ID: "1", Input: Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "2", Amount: "20", Expenses: "1"}},
		{ID: "2", Input: Input{Kind: "buy", Date: "2026-01-02", AssetID: "1", Quantity: "3", Amount: "30", Expenses: "0"}},
		{ID: "3", Input: Input{Kind: "sell", Date: "2026-01-02", AssetID: "1", Quantity: "5", Amount: "50", Expenses: "2"}},
		{ID: "99", Ticker: "SBER", Input: Input{Kind: "split", Date: "2026-01-02", AssetID: "1", OldUnits: "1", NewUnits: "10"}},
	}
	h, err := Rebuild(ops)
	if err != nil || h.Days[1].Quantities["1"] != "18" || h.Balance != "-3" || len(h.Warnings) != 0 || h.Splits["99"] != (SplitQuantities{"2", "20"}) {
		t.Fatal(h, err)
	}
	ops[3].NewUnits = "2"
	h, err = Rebuild(ops)
	if err != nil || h.Days[1].Quantities["1"] != "2" {
		t.Fatal(h, err)
	}
	ops[0].Quantity = "1"
	ops[3].OldUnits = "3"
	ops[3].NewUnits = "1"
	ops = append(ops, Operation{ID: "100", Ticker: "SBER", Input: Input{Kind: "split", Date: "2026-01-02", AssetID: "1", OldUnits: "1", NewUnits: "3"}})
	_, err = Rebuild(ops)
	var split *SplitError
	if !errors.As(err, &split) || split.OldUnits != "3" || !strings.Contains(err.Error(), "02.01.2026") || !strings.Contains(err.Error(), "SBER") {
		t.Fatal(err)
	}
	// Multiple same-day splits use numeric ID ASC before every trade.
	ops[0].Quantity = "3"
	h, err = Rebuild(ops)
	if err != nil || h.Splits["99"] != (SplitQuantities{"3", "1"}) || h.Splits["100"] != (SplitQuantities{"1", "3"}) || h.Days[1].Quantities["1"] != "1" {
		t.Fatal(h, err)
	}
}
