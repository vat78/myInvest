package sqlite

import (
	"context"
	"myinvest/internal/operations"
	operationssqlite "myinvest/internal/operations/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/internal/valuation"
	"myinvest/migrations"
	"path/filepath"
	"testing"
)

func TestCapitalHistoryUsesPersistedBalancesAndDirectRates(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "history.db"), migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	db := store.DB()
	if _, err = db.Exec(`INSERT INTO portfolios(id,name,name_key,opened_on,currency_code) VALUES
 (1,'R','r','2026-01-01','RUB'),(2,'U','u','2026-01-01','USD');
 INSERT INTO currency_rates(date,base,quote,value) VALUES
 ('2026-01-01','USD','RUB','90.125'),('2026-01-03','USD','RUB','100')`); err != nil {
		t.Fatal(err)
	}
	ops := operations.NewService(operationssqlite.NewRepository(db))
	for _, tc := range []struct{ pid, code, amount, date string }{
		{"1", "RUB", "0.125", "2026-01-01"},
		{"2", "USD", "1.25", "2026-01-01"},
		{"2", "USD", "1000", "2026-01-03"},
	} {
		if _, err = ops.Save(ctx, tc.pid, "", operations.Input{Kind: "deposit", Date: tc.date, CurrencyCode: tc.code, Amount: tc.amount, PortfolioAmount: tc.amount}); err != nil {
			t.Fatal(err)
		}
	}
	// An absent journal detail cannot affect this read: amounts come from daily snapshots.
	if _, err = db.Exec("DROP TABLE cash_flow_details"); err != nil {
		t.Fatal(err)
	}
	s := valuation.NewService(NewRepository(db))
	out, err := s.CapitalHistory(ctx, "RUB", "2026-01-01", "2026-01-02")
	if err != nil || len(out) != 2 {
		t.Fatal(out, err)
	}
	for _, p := range out {
		if !p.Complete || p.Value != "112.78125" {
			t.Fatal(p)
		}
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM operations").Scan(&count); err != nil || count != 3 {
		t.Fatal("history read changed journal", count, err)
	}
}
