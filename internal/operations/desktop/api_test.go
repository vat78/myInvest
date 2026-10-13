package desktop

import (
	"context"
	"errors"
	"myinvest/internal/operations"
	operationssqlite "myinvest/internal/operations/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"strings"
	"testing"
)

func TestBridge(t *testing.T) {
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "db.sqlite3"), migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = store.DB().Exec("INSERT INTO portfolios VALUES (1,'Test','test','2026-01-01','RUB')"); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(func(fn func(*operations.Service) error) error {
		return fn(operations.NewService(operationssqlite.NewRepository(store.DB())))
	})
	in := CashFlowInputDTO{Kind: "deposit", Date: "9999-12-31", CurrencyCode: "RUB", Amount: "0,1", PortfolioAmount: "0.100", Comment: "text"}
	for _, id := range []string{"0", "-1", "01", "1.0", "9223372036854775808"} {
		if got := api.SaveCashFlow(id, "", in); got.Message == "" {
			t.Fatal(got)
		}
		if _, err := api.ListCashFlows(id); err == nil {
			t.Fatal(id)
		}
	}
	got := api.SaveCashFlow("1", "", in)
	if got.Message != "" || len(got.Errors) > 0 {
		t.Fatal(got)
	}
	snapshot, err := api.ListCashFlows("1")
	if err != nil || snapshot.Balance != "0.1" || snapshot.Negative || snapshot.Operations[0].Rate != "1" || snapshot.Operations[0].PortfolioID != "1" {
		t.Fatal(snapshot, err)
	}
	id := snapshot.Operations[0].ID
	in.PortfolioAmount = "0.2"
	if got := api.SaveCashFlow("1", id, in); got.Errors["portfolioAmount"] == "" {
		t.Fatal(got)
	}
	in.Kind = "withdrawal"
	in.Amount = "0.2"
	in.PortfolioAmount = "0.2"
	if got := api.SaveCashFlow("1", id, in); got.Message != "" || len(got.Errors) > 0 {
		t.Fatal(got)
	}
	snapshot, err = api.ListCashFlows("1")
	if err != nil || snapshot.Balance != "-0.2" || !snapshot.Negative {
		t.Fatal(snapshot, err)
	}
	if got := api.DeleteCashFlow("1", id); got.Message != "" {
		t.Fatal(got)
	}
	if got := api.DeleteCashFlow("1", id); got.Message == "" {
		t.Fatal(got)
	}
	p := api.PreviewCashRate("3", "1")
	if p.Rate != "0.3333" || len(p.Errors) > 0 {
		t.Fatal(p)
	}
	p = api.PreviewCashRate("0", "oops")
	if len(p.Errors) != 2 {
		t.Fatal(p)
	}
}
func TestStorageErrorIsPrivate(t *testing.T) {
	api := NewAPI(func(func(*operations.Service) error) error { return errors.New("SQL private path") })
	got := api.SaveCashFlow("1", "", CashFlowInputDTO{})
	if got.Message == "" || strings.Contains(got.Message, "SQL") {
		t.Fatal(got)
	}
	if _, err := api.ListCashFlows("1"); err == nil || strings.Contains(err.Error(), "SQL") {
		t.Fatal(err)
	}
	got = api.DeleteCashFlow("1", "1")
	if got.Message == "" || strings.Contains(got.Message, "SQL") {
		t.Fatal(got)
	}
}

func TestTradeBridge(t *testing.T) {
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "db.sqlite3"), migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = store.DB().Exec("INSERT INTO portfolios VALUES(1,'Test','test','2026-01-01','RUB')"); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(func(fn func(*operations.Service) error) error {
		return fn(operations.NewService(operationssqlite.NewRepository(store.DB())))
	})
	asset := api.AddAsset(" sber ")
	if asset.Message != "" || len(asset.Errors) > 0 || asset.Asset.Ticker != "SBER" {
		t.Fatal(asset)
	}
	if got := api.AddAsset("SBER"); got.Errors["ticker"] == "" {
		t.Fatal(got)
	}
	preview := api.PreviewTrade("buy", "6", "1", "0")
	if preview.AveragePrice != "0.166666666666666667" || !preview.Rounded || preview.MoneyChange != "-1" {
		t.Fatal(preview)
	}
	small := api.PreviewTrade("buy", "1", "0.00000000000000000001", "0")
	if small.AveragePrice != "0" || !small.Rounded || small.MoneyChange != "-0.00000000000000000001" {
		t.Fatal(small)
	}
	if p := api.PreviewTrade("sell", "1", "120", "121"); p.MoneyChange != "-1" {
		t.Fatal(p)
	}
	in := TradeInputDTO{Kind: "sell", Date: "2026-01-01", AssetID: asset.Asset.ID, Quantity: "1", Amount: "120", Expenses: "2"}
	if got := api.SaveTrade("1", "", in); got.Message != "" || len(got.Errors) > 0 {
		t.Fatal(got)
	}
	snap, err := api.ListCashFlows("1")
	if err != nil || snap.Balance != "118" || len(snap.Warnings) != 1 || snap.Warnings[0].Available != "0" || snap.Operations[0].Ticker != "SBER" || snap.Operations[0].MoneyChange != "118" || snap.Operations[0].AveragePrice != "120" {
		t.Fatal(snap, err)
	}
	for _, id := range []string{"0", "01", "-1", "1.0", "9223372036854775808"} {
		if r := api.SaveTrade(id, "", in); r.Message == "" {
			t.Fatal(r)
		}
		if r := api.SaveTrade("1", id, in); r.Message == "" {
			t.Fatal(r)
		}
	}
	in.AssetID = "01"
	if r := api.SaveTrade("1", "", in); r.Errors["assetID"] == "" {
		t.Fatal(r)
	}
	in.AssetID = asset.Asset.ID
	in.Kind = "deposit"
	if r := api.SaveTrade("1", "", in); r.Errors["kind"] == "" {
		t.Fatal(r)
	}
	in.Kind = "buy"
	in.Quantity = "0"
	if r := api.SaveTrade("1", "", in); r.Errors["quantity"] == "" {
		t.Fatal(r)
	}
	assets, err := api.ListAssets()
	if err != nil || len(assets) != 1 {
		t.Fatal(assets, err)
	}
}
