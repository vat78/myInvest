package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"myinvest/internal/benchmarks"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestHistoryRecalculationPersistenceAndRollback(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "benchmarks.db")
	store, e := storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { store.Close() }()
	s := benchmarks.NewService(NewRepository(store.DB()))
	b, e := s.Create(ctx, benchmarks.Benchmark{Name: "Инфляция"})
	if e != nil {
		t.Fatal(e)
	}
	for _, in := range []benchmarks.Benchmark{{Name: " инфляция "}, {Name: "unknown", CurrencyCode: "XXX"}} {
		if _, e = s.Create(ctx, in); e == nil {
			t.Fatal("invalid directory accepted")
		}
	}
	if _, e = s.Create(ctx, benchmarks.Benchmark{Name: "Недвижимость", CurrencyCode: "RUB"}); e != nil {
		t.Fatal(e)
	}
	save := func(date, mode, input string) {
		t.Helper()
		if e = s.Save(ctx, benchmarks.Point{BenchmarkID: b.ID, Date: date, Mode: mode, Input: input}); e != nil {
			t.Fatal(e)
		}
	}
	check := func(values, growth []string) {
		t.Helper()
		rows, e := s.History(ctx, b.ID)
		if e != nil || len(rows) != len(values) {
			t.Fatal(rows, e)
		}
		for i, p := range rows {
			if p.Value != values[i] || p.Growth != growth[i] {
				t.Fatalf("row %d: %+v", i, p)
			}
		}
	}
	if e = s.Save(ctx, benchmarks.Point{BenchmarkID: b.ID, Date: "2026-01-01", Mode: "growth", Input: "5"}); e == nil {
		t.Fatal("initial growth")
	}
	save("2026-01-01", "value", "100")
	save("2026-03-01", "growth", "10")
	save("2026-04-01", "growth", "-5")
	check([]string{"100", "110", "104.5"}, []string{"", "10", "-5"})
	save("2026-01-01", "value", "200")
	check([]string{"200", "220", "209"}, []string{"", "10", "-5"})
	save("2026-02-01", "value", "100")
	check([]string{"200", "100", "110", "104.5"}, []string{"", "-50", "10", "-5"})
	save("2026-04-01", "value", "300")
	save("2026-02-01", "value", "200")
	check([]string{"200", "200", "220", "300"}, []string{"", "0", "10", "36.363636363636363636"})
	p, e := s.Preview(ctx, benchmarks.Point{BenchmarkID: b.ID, Date: "2026-03-01", Mode: "growth", Input: "20"})
	if e != nil || p.Value != "240" {
		t.Fatal(p, e)
	}
	if _, e = store.DB().Exec(`CREATE TRIGGER fail_later BEFORE UPDATE ON benchmark_points WHEN NEW.date='2026-04-01' BEGIN SELECT RAISE(ABORT,'private failure'); END`); e != nil {
		t.Fatal(e)
	}
	if e = s.Save(ctx, benchmarks.Point{BenchmarkID: b.ID, Date: "2026-01-01", Mode: "value", Input: "500"}); e == nil {
		t.Fatal("expected rollback")
	}
	check([]string{"200", "200", "220", "300"}, []string{"", "0", "10", "36.363636363636363636"})
	if e = s.Save(ctx, benchmarks.Point{BenchmarkID: "999", Date: "2026-01-01", Mode: "value", Input: "1"}); e == nil {
		t.Fatal("unknown reference")
	}
	for _, tc := range []struct{ id, date, mode, input, value, growth string }{
		{"999", "2025-01-01", "value", "1", "1", ""}, {b.ID, "2025-02-29", "value", "1", "1", ""}, {b.ID, "2025-01-01", "bad", "1", "1", ""},
		{b.ID, "2025-01-01", "value", "1", "0", ""}, {b.ID, "2025-01-01", "value", "1", "1e2", ""}, {b.ID, "2025-01-01", "growth", "1", "1", "2"},
		{b.ID, "2025-01-01", "growth", "-.1", "1", "-.1"},
		{b.ID, "2025-01-01", "growth", "-100", "1", "-100"},
		{b.ID, "2025-01-01", "growth", "-1000", "1", "-1000"},
		{b.ID, "2025-01-01", "growth", "1", "1", ""},
		{b.ID, "2025-01-01", "value", "2", "1", ""},
	} {
		if _, e = store.DB().Exec(`INSERT INTO benchmark_points VALUES(?,?,?,?,?,NULLIF(?,''))`, tc.id, tc.date, tc.mode, tc.input, tc.value, tc.growth); e == nil {
			t.Fatalf("constraint accepted %+v", tc)
		}
	}
	store.Close()
	store, e = storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	s = benchmarks.NewService(NewRepository(store.DB()))
	check([]string{"200", "200", "220", "300"}, []string{"", "0", "10", "36.363636363636363636"})
	rows, e := s.List(ctx)
	if e != nil || len(rows) != 2 {
		t.Fatal(rows, e)
	}
}

type previousBundle struct{ fstest.MapFS }

func (previousBundle) AfterMigration(ctx context.Context, v int, tx *sql.Tx) error {
	return migrations.Files.AfterMigration(ctx, v, tx)
}
func TestUpgradeBacksUpExistingDatabase(t *testing.T) {
	ctx := context.Background()
	old := previousBundle{fstest.MapFS{}}
	names, e := fs.ReadDir(migrations.Files, ".")
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range names {
		if name.Name() >= "0012" {
			continue
		}
		data, e := fs.ReadFile(migrations.Files, name.Name())
		if e != nil {
			t.Fatal(e)
		}
		old.MapFS[name.Name()] = &fstest.MapFile{Data: data}
	}
	path := filepath.Join(t.TempDir(), "upgrade.db")
	store, e := storage.Open(ctx, path, old)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.DB().Exec(`INSERT INTO assets(id,ticker,type_code) VALUES(1,'AAA','stocks'); INSERT INTO asset_quotes VALUES(1,'2026-01-01','RUB','12.123456789012345678900')`); e != nil {
		t.Fatal(e)
	}
	store.Close()
	store, e = storage.Open(ctx, path, migrations.Files)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	var price string
	if e = store.DB().QueryRow(`SELECT price FROM asset_quotes`).Scan(&price); e != nil || price != "12.123456789012345678900" {
		t.Fatal(price, e)
	}
	rows, e := NewRepository(store.DB()).List(ctx)
	if e != nil || len(rows) != 0 {
		t.Fatal(rows, e)
	}
	backups, e := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if e != nil || len(backups) != 1 {
		t.Fatal(backups, e)
	}
	backup, e := sql.Open("sqlite", backups[0])
	if e != nil {
		t.Fatal(e)
	}
	defer backup.Close()
	var versions int
	if e = backup.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&versions); e != nil || versions != 11 {
		t.Fatal(versions, e)
	}
	if e = backup.QueryRow(`SELECT price FROM asset_quotes`).Scan(&price); e != nil || price != "12.123456789012345678900" {
		t.Fatal(price, e)
	}
}
