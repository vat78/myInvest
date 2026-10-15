package sqlite_test

import (
	"myinvest/internal/operations"
	"myinvest/internal/valuation"
	valuationsqlite "myinvest/internal/valuation/sqlite"
	"testing"
)

func TestAnalysisSnapshotCRUDAndIsolation(t *testing.T) {
	store, s, _ := setup(t)
	asset, err := s.AddAsset(ctx, "A", "stocks")
	if err != nil {
		t.Fatal(err)
	}
	save := func(pid, id, kind, q, amount, date string) operations.Operation {
		t.Helper()
		op, err := s.Save(ctx, pid, id, trade(kind, asset.ID, q, amount, "5", date))
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	buy := save("1", "", "buy", "2", "200", "2026-01-01")
	save("1", "", "buy", "1", "160", "2026-01-02")
	sale := save("1", "", "sell", "1", "999", "2026-01-03")
	save("1", "", "buy", "1", "9999", "2027-01-01")
	save("2", "", "buy", "1", "500", "2026-01-01")
	v := valuation.NewService(valuationsqlite.NewRepository(store.DB()))
	check := func(pid, current, average string) {
		t.Helper()
		out, err := v.Analysis(ctx, pid, "2026-10-14")
		if err != nil || len(out.Assets) != 1 || out.Assets[0].CurrentValue != current || out.Assets[0].AveragePrice != average {
			t.Fatal(out, err)
		}
	}
	check("1", "1998", "130")
	check("2", "500", "500")
	save("1", buy.ID, "buy", "2", "260", "2026-01-01")
	check("1", "1998", "145")
	save("1", sale.ID, "sell", "2", "999", "2026-01-03")
	check("1", "499.5", "160")
	save("1", sale.ID, "sell", "1", "999", "2026-01-03")
	check("1", "1998", "145")
	if err = s.Delete(ctx, "1", sale.ID); err != nil {
		t.Fatal(err)
	}
	check("1", "480", "140")
	split, err := s.Save(ctx, "1", "", operations.Input{Kind: "split", Date: "2026-01-04", AssetID: asset.ID, OldUnits: "1", NewUnits: "2"})
	if err != nil {
		t.Fatal(err)
	}
	check("1", "480", "70")
	if err = s.Delete(ctx, "1", split.ID); err != nil {
		t.Fatal(err)
	}
	check("1", "480", "140")
	before := scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-02'")
	if _, err = v.Analysis(ctx, "1", "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if after := scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-02'"); before != after {
		t.Fatal("analysis mutated data")
	}
	if _, err = v.Analysis(ctx, "999", "2026-10-14"); err == nil {
		t.Fatal("missing portfolio accepted")
	}
	if _, err = store.DB().Exec("DELETE FROM trade_details WHERE operation_id=?", buy.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = v.Analysis(ctx, "1", "2026-10-14"); err == nil {
		t.Fatal("incomplete history accepted")
	}
}
