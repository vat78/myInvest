package desktop

import (
	"context"
	"errors"
	"myinvest/internal/cashflow"
	cashsqlite "myinvest/internal/cashflow/sqlite"
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
	api := NewAPI(func(fn func(*cashflow.Service) error) error {
		return fn(cashflow.NewService(cashsqlite.NewRepository(store.DB())))
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
	if p.Rate != "0.333333333333333333" || len(p.Errors) > 0 {
		t.Fatal(p)
	}
	p = api.PreviewCashRate("0", "oops")
	if len(p.Errors) != 2 {
		t.Fatal(p)
	}
}
func TestStorageErrorIsPrivate(t *testing.T) {
	api := NewAPI(func(func(*cashflow.Service) error) error { return errors.New("SQL private path") })
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
