package operations

import (
	"math/big"
	"testing"
)

func TestPersistedPositionValues(t *testing.T) {
	ops := []Operation{
		{ID: "1", Input: Input{Kind: "buy", Date: "2026-01-01", AssetID: "1", Quantity: "12", Amount: "12", Expenses: "0"}},
		{ID: "2", Input: Input{Kind: "split", Date: "2026-01-02", AssetID: "1", OldUnits: "1", NewUnits: "3"}},
		{ID: "3", Input: Input{Kind: "sell", Date: "2026-01-03", AssetID: "1", Quantity: "1", Amount: "2", Expenses: "0.1"}},
		{ID: "4", Input: Input{Kind: "split", Date: "2026-01-03", AssetID: "1", OldUnits: "10", NewUnits: "1"}},
	}
	h, err := Rebuild(ops)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"12", "12", "26/5"} {
		v := h.Days[i].Values["1"]
		r, ok := new(big.Rat).SetString(v.Numerator + "/" + v.Denominator)
		if !ok || r.RatString() != want {
			t.Fatal(i, v, r, want)
		}
	}
	if h.Days[1].Values["1"].PriceOperationID != "1" || h.Days[2].Values["1"].PriceOperationID != "3" {
		t.Fatal(h)
	}
	// The inverse price after a split may be periodic; never truncate it.
	ops = ops[:2]
	ops = append(ops, Operation{ID: "3", Input: Input{Kind: "sell", Date: "2026-01-03", AssetID: "1", Quantity: "1", Amount: "0.1", Expenses: "0"}})
	h, err = Rebuild(ops)
	if err != nil || h.Days[1].Values["1"].Numerator != "12" || h.Days[1].Values["1"].Denominator != "1" {
		t.Fatal(h, err)
	}
}
