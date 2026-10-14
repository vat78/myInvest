package sqlite

import (
	"context"
	"encoding/json"
	"myinvest/internal/benchmarks"
	benchmarkssqlite "myinvest/internal/benchmarks/sqlite"
	"myinvest/internal/planning"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"testing"
)

func TestPersistenceRecalculationAndRollback(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "plan.db")
	db, e := storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { db.Close() }()
	b := benchmarks.NewService(benchmarkssqlite.NewRepository(db.DB()))
	index, e := b.Create(ctx, benchmarks.Benchmark{Name: "Индекс", CurrencyCode: "RUB"})
	if e != nil {
		t.Fatal(e)
	}
	savePoint := func(date, value string) {
		t.Helper()
		if e := b.Save(ctx, benchmarks.Point{BenchmarkID: index.ID, Date: date, Mode: "value", Input: value}); e != nil {
			t.Fatal(e)
		}
	}
	savePoint("2023-12-31", "100")
	savePoint("2024-12-31", "110")
	s := planning.NewService(NewRepository(db.DB()))
	p := planning.Parameters{CurrencyCode: "RUB", StartYear: "2024", Years: "2", Mode: "goal", Goal: "1000", Contribution: "0", ReturnPercent: "0", InflationPercent: "0", BenchmarkID: index.ID, AllowEstimated: true, InitialCapital: "0"}
	out, e := s.Preview(ctx, p, "2026-10-14")
	if e != nil || out.Found || !out.Current.Complete {
		t.Fatal(out, e)
	}
	missing, e := s.Get(ctx, "RUB", "2026-10-14")
	if e != nil || missing.Found {
		t.Fatal(missing, e)
	}
	saved, e := s.Save(ctx, p, "2026-10-14")
	if e != nil || saved.Parameters.Revision != "1" {
		t.Fatal(saved, e)
	}
	original, _ := json.Marshal(saved.Original)
	savePoint("2024-12-31", "120")
	current, e := s.Get(ctx, "RUB", "2026-10-14")
	if e != nil || current.Current.FirstContribution == saved.Current.FirstContribution {
		t.Fatal(current, e)
	}
	same, _ := json.Marshal(current.Original)
	if string(original) != string(same) {
		t.Fatal("original overwritten")
	}
	if _, e = s.Save(ctx, p, "2026-10-14"); e == nil {
		t.Fatal("stale write accepted")
	}
	p = current.Parameters
	p.Mode = "contribution"
	p.Contribution = "123.4567890123456789"
	fixed, e := s.Save(ctx, p, "2026-10-14")
	if e != nil {
		t.Fatal(e)
	}
	savePoint("2024-12-31", "130")
	current, e = s.Get(ctx, "RUB", "2026-10-14")
	if e != nil || current.Current.FirstContribution != fixed.Current.FirstContribution || current.Current.FinalCapital == fixed.Current.FinalCapital {
		t.Fatal(current, e)
	}
	if _, e = db.DB().Exec(`CREATE TRIGGER reject_plan BEFORE UPDATE ON currency_financial_plans BEGIN SELECT RAISE(ABORT,'test'); END`); e != nil {
		t.Fatal(e)
	}
	p = current.Parameters
	p.Contribution = "999"
	if _, e = s.Save(ctx, p, "2026-10-14"); e == nil {
		t.Fatal("expected rollback")
	}
	after, e := s.Get(ctx, "RUB", "2026-10-14")
	if e != nil || after.Parameters.Contribution != current.Parameters.Contribution || after.Parameters.Revision != current.Parameters.Revision {
		t.Fatal(after, e)
	}
	var count int
	if e = db.DB().QueryRow("SELECT count(*) FROM operations").Scan(&count); e != nil || count != 0 {
		t.Fatal("operations changed", count, e)
	}
	if _, e = db.DB().Exec(`UPDATE currency_financial_plans SET benchmark_id=999`); e == nil {
		t.Fatal("foreign key accepted")
	}
	if _, e = db.DB().Exec(`UPDATE currency_financial_plans SET initial_denominator='0'`); e == nil {
		t.Fatal("zero denominator accepted")
	}
	if _, e = db.DB().Exec(`UPDATE currency_financial_plans SET years=0`); e == nil {
		t.Fatal("invalid horizon accepted")
	}
	for _, query := range []string{`UPDATE currency_financial_plans SET return_percent='-100'`, `UPDATE currency_financial_plans SET inflation_percent='-100.01'`, `UPDATE currency_financial_plans SET contribution='1e2'`, `UPDATE currency_financial_plans SET initial_capital='1/3'`, `UPDATE currency_financial_plans SET original='invalid'`} {
		if _, e = db.DB().Exec(query); e == nil {
			t.Fatal("invalid storage accepted", query)
		}
	}
	db.Close()
	db, e = storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	s = planning.NewService(NewRepository(db.DB()))
	out, e = s.Get(ctx, "RUB", "2026-10-14")
	if e != nil || out.Parameters.Contribution != current.Parameters.Contribution {
		t.Fatal(out, e)
	}
}

func TestPlansAreIndependentByCurrencyWithoutPortfolios(t *testing.T) {
	ctx := context.Background()
	db, e := storage.Open(ctx, filepath.Join(t.TempDir(), "currency.db"), migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	b := benchmarks.NewService(benchmarkssqlite.NewRepository(db.DB()))
	index, e := b.Create(ctx, benchmarks.Benchmark{Name: "Инфляция", CurrencyCode: "RUB"})
	if e != nil {
		t.Fatal(e)
	}
	s := planning.NewService(NewRepository(db.DB()))
	p := planning.Parameters{CurrencyCode: "RUB", StartYear: "2026", Years: "1", Mode: "contribution", Contribution: "12.1234567890123456789", ReturnPercent: "0", InflationPercent: "0", BenchmarkID: index.ID, InitialCapital: "10.1234567890123456789"}
	rub, e := s.Save(ctx, p, "2026-10-14")
	if e != nil {
		t.Fatal(e)
	}
	p.CurrencyCode = "USD"
	p.Contribution = "100"
	usd, e := s.Save(ctx, p, "2026-10-14")
	if e != nil {
		t.Fatal(e)
	}
	if rub.Currency != "RUB" || usd.Currency != "USD" || rub.Current.FinalCapital == usd.Current.FinalCapital {
		t.Fatal(rub, usd)
	}
	loaded, e := s.Get(ctx, " rub ", "2026-10-14")
	if e != nil || loaded.Parameters.InitialCapital != p.InitialCapital || loaded.Current.FirstContribution != rub.Current.FirstContribution {
		t.Fatal(loaded, e)
	}
	p.CurrencyCode = "XXX"
	if _, e = s.Save(ctx, p, "2026-10-14"); e == nil {
		t.Fatal("unknown currency accepted")
	}
	p.CurrencyCode = "RUB"
	p.BenchmarkID = "999"
	if _, e = s.Preview(ctx, p, "2026-10-14"); e == nil {
		t.Fatal("unknown benchmark accepted")
	}
	var count int
	if e = db.DB().QueryRow("SELECT count(*) FROM portfolios").Scan(&count); e != nil || count != 0 {
		t.Fatal("portfolio dependency", count, e)
	}
}
