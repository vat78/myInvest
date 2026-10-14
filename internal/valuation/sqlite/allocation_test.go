package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"myinvest/internal/operations"
	operationssqlite "myinvest/internal/operations/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/internal/valuation"
	"myinvest/migrations"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestAllocationPersistenceConflictAndRollback(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "allocation.db")
	store, e := storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { store.Close() }()
	db := store.DB()
	if _, e = db.Exec(`INSERT INTO portfolios(id,name,name_key,opened_on,currency_code) VALUES(1,'One','one','2026-01-01','RUB'),(2,'Two','two','2026-01-01','USD'); INSERT INTO assets(id,ticker,type_code) VALUES(1,'A','stocks'),(2,'B','stocks')`); e != nil {
		t.Fatal(e)
	}
	repo := NewRepository(db)
	service := valuation.NewAllocationService(repo)
	p := valuation.AllocationPlan{CashPercent: "20", Targets: []valuation.Target{{AssetID: "1", Percent: "40"}, {AssetID: "2", Percent: "40"}}}
	if e = service.Save(ctx, "1", p); e != nil {
		t.Fatal(e)
	}
	out, e := service.Load(ctx, "1", "2026-10-14")
	if e != nil || out.Plan.Revision != "1" || len(out.Plan.Targets) != 2 || !out.CanCalculate {
		t.Fatal(out, e)
	}
	original := out.Plan
	other, e := service.Load(ctx, "2", "2026-10-14")
	if e != nil || other.Plan.Generation != "" {
		t.Fatal(other, e)
	}
	if e = service.Save(ctx, "1", p); !errors.Is(e, valuation.ErrAllocationConflict) {
		t.Fatal(e)
	}
	invalid := original
	invalid.Targets = []valuation.Target{{AssetID: "999", Percent: "80"}}
	if e = service.Save(ctx, "1", invalid); e == nil {
		t.Fatal("unknown asset")
	}
	out, e = service.Load(ctx, "1", "2026-10-14")
	if e != nil || out.Plan.Revision != "1" || len(out.Plan.Targets) != 2 {
		t.Fatal("partial write", out, e)
	}
	if _, e = db.Exec(`CREATE TRIGGER reject_target BEFORE INSERT ON portfolio_allocation_targets WHEN NEW.asset_id=2 BEGIN SELECT RAISE(ABORT,'private failure'); END`); e != nil {
		t.Fatal(e)
	}
	if e = service.Save(ctx, "1", original); e == nil {
		t.Fatal("trigger accepted")
	}
	out, e = service.Load(ctx, "1", "2026-10-14")
	if e != nil || out.Plan.Revision != "1" || len(out.Plan.Targets) != 2 {
		t.Fatal("rollback", out, e)
	}
	db.Exec(`DROP TRIGGER reject_target`)
	if e = service.Save(ctx, "1", original); e != nil {
		t.Fatal(e)
	}
	if e = service.Save(ctx, "1", original); !errors.Is(e, valuation.ErrAllocationConflict) {
		t.Fatal(e)
	}
	if e = service.Delete(ctx, "1", original.Generation, original.Revision); !errors.Is(e, valuation.ErrAllocationConflict) {
		t.Fatal(e)
	}
	out, e = service.Load(ctx, "1", "2026-10-14")
	if e != nil {
		t.Fatal(e)
	}
	if e = service.Delete(ctx, "1", out.Plan.Generation, out.Plan.Revision); e != nil {
		t.Fatal(e)
	}
	if e = service.Save(ctx, "1", p); e != nil {
		t.Fatal(e)
	}
	if e = service.Save(ctx, "1", original); !errors.Is(e, valuation.ErrAllocationConflict) {
		t.Fatal("stale after recreation", e)
	}
	for _, query := range []string{`UPDATE portfolio_allocation_plans SET cash_weight_units=10001`, `UPDATE portfolio_allocation_targets SET weight_units=-1`, `INSERT INTO portfolio_allocation_targets VALUES(1,1,0)`, `INSERT INTO portfolio_allocation_targets VALUES(1,999,1)`} {
		if _, e = db.Exec(query); e == nil {
			t.Fatal("constraint accepted", query)
		}
	}
	store.Close()
	store, e = storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	out, e = valuation.NewAllocationService(NewRepository(store.DB())).Load(ctx, "1", "2026-10-14")
	if e != nil || len(out.Plan.Targets) != 2 {
		t.Fatal(out, e)
	}
	if e = valuation.NewAllocationService(NewRepository(store.DB())).Save(ctx, "999", p); e == nil {
		t.Fatal("unknown portfolio")
	}
}

type beforeAllocation struct{ fstest.MapFS }

func (beforeAllocation) AfterMigration(ctx context.Context, v int, tx *sql.Tx) error {
	return migrations.Files.AfterMigration(ctx, v, tx)
}
func TestAllocationMigrationPreservesDataAndBacksUp(t *testing.T) {
	ctx := context.Background()
	old := beforeAllocation{fstest.MapFS{}}
	entries, e := fs.ReadDir(migrations.Files, ".")
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if entry.Name() >= "0011" {
			continue
		}
		b, e := fs.ReadFile(migrations.Files, entry.Name())
		if e != nil {
			t.Fatal(e)
		}
		old.MapFS[entry.Name()] = &fstest.MapFile{Data: b}
	}
	path := filepath.Join(t.TempDir(), "upgrade.db")
	store, e := storage.Open(ctx, path, old)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.DB().Exec(`INSERT INTO assets(id,ticker,type_code) VALUES(1,'A','stocks'); INSERT INTO asset_quotes VALUES(1,'2026-10-14','RUB','123.4567890123456789')`); e != nil {
		t.Fatal(e)
	}
	store.Close()
	store, e = storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	var price string
	if e = store.DB().QueryRow(`SELECT price FROM asset_quotes`).Scan(&price); e != nil || price != "123.4567890123456789" {
		t.Fatal(price, e)
	}
	backups, e := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if e != nil || len(backups) != 1 {
		t.Fatal(backups, e)
	}
}

func TestAllocationTracksLedgerQuotesAndDirectFX(t *testing.T) {
	ctx := context.Background()
	store, e := storage.Open(ctx, filepath.Join(t.TempDir(), "snapshot.db"), migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	db := store.DB()
	if _, e = db.Exec(`INSERT INTO portfolios(id,name,name_key,opened_on,currency_code) VALUES(1,'One','one','2026-01-01','RUB');INSERT INTO assets(id,ticker,type_code) VALUES(1,'A','stocks'),(2,'B','stocks')`); e != nil {
		t.Fatal(e)
	}
	ledger := operations.NewService(operationssqlite.NewRepository(db))
	for _, in := range []operations.Input{{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1200", PortfolioAmount: "1200"}, {Kind: "buy", Date: "2026-01-02", AssetID: "1", Quantity: "6", Amount: "600", Expenses: "0"}, {Kind: "buy", Date: "2026-01-02", AssetID: "2", Quantity: "3", Amount: "300", Expenses: "0"}} {
		if _, e = ledger.Save(ctx, "1", "", in); e != nil {
			t.Fatal(e)
		}
	}
	service := valuation.NewAllocationService(NewRepository(db))
	if e = service.Save(ctx, "1", valuation.AllocationPlan{CashPercent: "20", Targets: []valuation.Target{{AssetID: "1", Percent: "40"}, {AssetID: "2", Percent: "40"}}}); e != nil {
		t.Fatal(e)
	}
	out, e := service.Load(ctx, "1", "2026-10-14")
	if e != nil || out.Rows[0].CurrentValue != "600" || out.Rows[1].BuyValue != "60" {
		t.Fatal(out, e)
	}
	if _, e = db.Exec(`INSERT INTO asset_quotes VALUES(1,'2026-01-03','USD','2');INSERT INTO currency_rates(date,base,quote,value) VALUES('2026-01-03','USD','RUB','100')`); e != nil {
		t.Fatal(e)
	}
	out, e = service.Load(ctx, "1", "2026-10-14")
	if e != nil || out.Total != "1800" || out.Rows[0].CurrentValue != "1200" || out.Rows[0].PriceRate != "100" || out.Reserve != "360" || out.ReserveShortfall != "60" {
		t.Fatal(out, e)
	}
	// The new sale affects cash/quantity, but never supersedes the purchase/quote source.
	if _, e = ledger.Save(ctx, "1", "", operations.Input{Kind: "sell", Date: "2026-01-04", AssetID: "1", Quantity: "1", Amount: "999", Expenses: "1"}); e != nil {
		t.Fatal(e)
	}
	out, e = service.Load(ctx, "1", "2026-10-14")
	if e != nil || out.Rows[0].CurrentValue != "1000" || out.Rows[0].PriceSource != "quote" || out.Cash != "1298" {
		t.Fatal(out, e)
	}
}
