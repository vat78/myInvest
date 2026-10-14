package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"myinvest/internal/quotes"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestQuotesLifecycleAndConstraints(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "quotes.db")
	store, err := storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { store.Close() }()
	db := store.DB()
	repo := NewRepository(db)
	s := quotes.NewService(repo)
	rows, err := s.List(ctx)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty: %v %v", rows, err)
	}
	if _, err = db.Exec(`INSERT INTO assets(id,ticker,type_code) VALUES(1,'AAA','stocks'),(2,'BBB','stocks')`); err != nil {
		t.Fatal(err)
	}
	price := "12.1234567890123456789012345678900"
	q := quotes.Quote{AssetID: "1", Date: "2026-10-14", CurrencyCode: "RUB", Price: price}
	if err = s.Save(ctx, q); err != nil {
		t.Fatal(err)
	}
	q.Price = "0.2"
	if err = s.Save(ctx, q); err != nil {
		t.Fatal(err)
	}
	q.Price = price
	if err = s.Save(ctx, q); err != nil {
		t.Fatal(err)
	}
	for _, other := range []quotes.Quote{{AssetID: "1", Date: "2026-10-13", CurrencyCode: "RUB", Price: "1"}, {AssetID: "1", Date: q.Date, CurrencyCode: "USD", Price: "2"}, {AssetID: "2", Date: q.Date, CurrencyCode: "RUB", Price: "3"}} {
		if err = s.Save(ctx, other); err != nil {
			t.Fatal(err)
		}
	}
	rows, err = s.List(ctx)
	if err != nil || len(rows) != 4 || rows[0].Price != price || rows[0].Ticker != "AAA" || rows[1].CurrencyCode != "USD" || rows[3].Date != "2026-10-13" {
		t.Fatalf("ordered upsert: %#v %v", rows, err)
	}
	for _, tc := range []struct{ asset, date, code, price string }{
		{"999", q.Date, "RUB", "1"}, {"1", q.Date, "XXX", "1"}, {"1", "2026-02-29", "RUB", "1"}, {"1", "0000-01-01", "RUB", "1"},
		{"1", q.Date, "RUB", "0"}, {"1", q.Date, "RUB", "-1"}, {"1", q.Date, "RUB", "1e2"}, {"1", q.Date, "RUB", ".1"}, {"1", q.Date, "RUB", "1."}, {"1", q.Date, "RUB", "1.2.3"},
	} {
		if _, err = db.Exec(`INSERT INTO asset_quotes(asset_id,date,currency_code,price) VALUES(?,?,?,?)`, tc.asset, tc.date, tc.code, tc.price); err == nil {
			t.Fatalf("constraint accepted %#v", tc)
		}
	}
	for _, in := range []quotes.Quote{{AssetID: "999", Date: q.Date, CurrencyCode: "RUB", Price: "1"}, {AssetID: "1", Date: q.Date, CurrencyCode: "XXX", Price: "1"}} {
		if err = s.Save(ctx, in); err == nil {
			t.Fatal("unknown reference accepted")
		}
	}
	if _, err = db.Exec(`CREATE TRIGGER reject_quote BEFORE UPDATE ON asset_quotes BEGIN SELECT RAISE(ABORT,'private failure'); END`); err != nil {
		t.Fatal(err)
	}
	q.Price = "99"
	if err = s.Save(ctx, q); err == nil {
		t.Fatal("expected write failure")
	}
	rows, err = s.List(ctx)
	if err != nil || rows[0].Price != price {
		t.Fatalf("failed write changed price: %#v %v", rows, err)
	}
	store.Close()
	store, err = storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	rows, err = NewRepository(store.DB()).List(ctx)
	if err != nil || len(rows) != 4 || rows[0].Price != price {
		t.Fatalf("reopen: %#v %v", rows, err)
	}
}

// Keep the same Go migration hook while emulating the previous released schema.
type previousBundle struct{ fstest.MapFS }

func (previousBundle) AfterMigration(ctx context.Context, v int, tx *sql.Tx) error {
	return migrations.Files.AfterMigration(ctx, v, tx)
}
func TestUpgradePreservesDirectoryAndRates(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "upgrade.db")
	old := previousBundle{fstest.MapFS{}}
	names, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name.Name() >= "0010" {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, name.Name())
		if err != nil {
			t.Fatal(err)
		}
		old.MapFS[name.Name()] = &fstest.MapFile{Data: data}
	}
	store, err := storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB().Exec(`INSERT INTO assets(id,ticker,type_code) VALUES(1,'AAA','stocks'); INSERT INTO currency_rates(date,base,quote,value) VALUES('2026-10-14','USD','RUB','95.123456789012345678900')`); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var ticker, value string
	if err = store.DB().QueryRow("SELECT ticker FROM assets WHERE id=1").Scan(&ticker); err != nil || ticker != "AAA" {
		t.Fatalf("asset lost: %s %v", ticker, err)
	}
	if err = store.DB().QueryRow("SELECT value FROM currency_rates WHERE base='USD' AND quote='RUB'").Scan(&value); err != nil || value != "95.123456789012345678900" {
		t.Fatalf("rate lost: %s %v", value, err)
	}
	rows, err := NewRepository(store.DB()).List(ctx)
	if err != nil || len(rows) != 0 {
		t.Fatalf("migration invented quotes: %v %v", rows, err)
	}
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backup: %v %v", backups, err)
	}
	backup, err := sql.Open("sqlite", backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var versions int
	if err = backup.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&versions); err != nil || versions != 9 {
		t.Fatalf("backup schema: %d %v", versions, err)
	}
	if err = backup.QueryRow("SELECT value FROM currency_rates WHERE base='USD'").Scan(&value); err != nil || value != "95.123456789012345678900" {
		t.Fatalf("backup data: %s %v", value, err)
	}
}
