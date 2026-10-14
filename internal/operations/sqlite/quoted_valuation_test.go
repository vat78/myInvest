package sqlite_test

import (
	"myinvest/internal/operations"
	"myinvest/internal/quotes"
	quotessqlite "myinvest/internal/quotes/sqlite"
	"myinvest/internal/valuation"
	valuationsqlite "myinvest/internal/valuation/sqlite"
	"testing"
)

func TestQuoteValuationHistoryAndPersistence(t *testing.T) {
	store, s, _ := setup(t)
	asset, err := s.AddAsset(ctx, "A", "stocks")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", "", operations.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1000", PortfolioAmount: "1000"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", "", trade("buy", asset.ID, "2", "200", "0", "2026-01-01")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "2", "", trade("buy", asset.ID, "1", "10", "0", "2026-01-01")); err != nil {
		t.Fatal(err)
	}
	qs := quotes.NewService(quotessqlite.NewRepository(store.DB()))
	v := valuation.NewService(valuationsqlite.NewRepository(store.DB()))
	saveQuote := func(date, code, price string) {
		t.Helper()
		if err := qs.Save(ctx, quotes.Quote{AssetID: asset.ID, Date: date, CurrencyCode: code, Price: price}); err != nil {
			t.Fatal(err)
		}
	}
	saveQuote("2026-01-02", "RUB", "120")
	if _, err = store.DB().Exec(`INSERT INTO currency_rates VALUES('2026-01-01','RUB','USD','0.01'),('2026-01-01','USD','RUB','100')`); err != nil {
		t.Fatal(err)
	}
	check := func(pid, date, current, average string) {
		t.Helper()
		out, err := v.Analysis(ctx, pid, date)
		if err != nil || len(out.Assets) != 1 || out.Assets[0].CurrentValue != current || out.Assets[0].AveragePrice != average {
			t.Fatal(out, err)
		}
	}
	check("1", "2026-01-02", "240", "100")
	check("2", "2026-01-02", "1.2", "10")
	if _, err = s.Save(ctx, "1", "", trade("sell", asset.ID, "1", "150", "0", "2026-01-03")); err != nil {
		t.Fatal(err)
	}
	check("1", "2026-01-03", "150", "100")
	saveQuote("2026-01-04", "USD", "2")
	check("1", "2026-01-04", "200", "100")
	saveQuote("2026-01-04", "USD", "3")
	check("1", "2026-01-04", "300", "100")
	saveQuote("2027-01-01", "RUB", "999")
	check("1", "2026-01-04", "300", "100")
	if _, err = s.Save(ctx, "1", "", operations.Input{Kind: "split", Date: "2026-01-05", AssetID: asset.ID, OldUnits: "1", NewUnits: "2"}); err != nil {
		t.Fatal(err)
	}
	check("1", "2026-01-05", "300", "50")
	before := scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-05'")
	out, err := v.Overview(ctx, "2026-01-05")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range out.Portfolios {
		if p.ID == "1" && p.Summary.Value != "1250" {
			t.Fatalf("RUB card: %#v", p)
		}
	}
	if scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-05'") != before {
		t.Fatal("quote changed stored trade values")
	}
	// Overview uses persisted quantities and trade values, never trade details.
	if _, err = store.DB().Exec("DROP TABLE trade_details"); err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-05")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range out.Portfolios {
		if p.ID == "1" && p.Summary.Value != "1250" {
			t.Fatal(p)
		}
	}
}
