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

func TestComparisonSnapshot(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "comparison.db"), migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	db := store.DB()
	_, err = db.Exec(`INSERT INTO portfolios(id,name,name_key,opened_on,currency_code) VALUES(1,'One','one','2025-01-01','USD');
 INSERT INTO benchmarks(id,name,name_key,currency_code) VALUES(1,'Inflation','inflation',NULL);
 INSERT INTO benchmark_points(benchmark_id,date,mode,input,value) VALUES(1,'2024-12-31','value','100','100'),(1,'2026-12-31','value','110','110');
 INSERT INTO currency_rates(date,base,quote,value) VALUES('2025-01-01','USD','RUB','90.125')`)
	if err != nil {
		t.Fatal(err)
	}
	s := operations.NewService(operationssqlite.NewRepository(db))
	for _, in := range []operations.Input{
		{Kind: "deposit", Date: "2025-01-01", CurrencyCode: "USD", Amount: "100.125", PortfolioAmount: "100.125"},
		{Kind: "withdrawal", Date: "2025-01-02", CurrencyCode: "USD", Amount: "5.005", PortfolioAmount: "5.005"},
		{Kind: "dividend", Date: "2025-12-31", Amount: "9.512"},
		{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "USD", Amount: "20", PortfolioAmount: "20"},
	} {
		if _, err = s.Save(ctx, "1", "", in); err != nil {
			t.Fatal(err)
		}
	}
	out, err := NewRepository(db).LoadComparison(ctx, "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Flows) != 2 || out.Flows[0].Amount != "100.125" || out.Flows[1].Kind != "withdrawal" || out.Flows[1].Amount != "5.005" {
		t.Fatal(out.Flows)
	}
	if len(out.Benchmarks) != 1 || len(out.Benchmarks[0].Points) != 1 || out.Benchmarks[0].Points[0].Value != "100" {
		t.Fatal(out.Benchmarks)
	}
	if len(out.Rates) != 1 || out.Rates[0].Value != "90.125" || len(out.Portfolios) != 1 {
		t.Fatal(out)
	}
	if got := out.Portfolios[0].Days[len(out.Portfolios[0].Days)-1].Cash; got != "104.632" {
		t.Fatal(got)
	}
	comparison, err := valuation.NewComparisonService(NewRepository(db)).Compare(ctx, "1", "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	p := comparison.Series[1].Points[1]
	if p.Annual != "10" || p.Cumulative != "10" {
		t.Fatal(p)
	}
}
