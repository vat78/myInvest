package sqlite

import (
	"context"
	"errors"
	"io/fs"
	"myinvest/internal/portfolio"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
)

var ctx = context.Background()

func open(t *testing.T, path string, files fs.FS) *storage.Store {
	t.Helper()
	s, err := storage.Open(ctx, path, files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
func TestLifecycleAndRejectedWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "portfolios.sqlite3")
	store := open(t, path, migrations.Files)
	r := NewRepository(store.DB())
	s := portfolio.NewService(r)
	rows, err := s.List(ctx)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("initial: %v %v", rows, err)
	}
	a, err := s.Create(ctx, portfolio.Input{Name: " Основной ", OpenedOn: "2024-02-29", CurrencyCode: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(ctx, portfolio.Input{Name: "Alpha", OpenedOn: "2099-12-31", CurrencyCode: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Create(ctx, portfolio.Input{Name: "Beta", OpenedOn: "2099-12-31", CurrencyCode: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.List(ctx)
	if !reflect.DeepEqual([]string{before[0].ID, before[1].ID, before[2].ID}, []string{b.ID, c.ID, a.ID}) {
		t.Fatalf("ordering: %+v", before)
	}
	for _, input := range []portfolio.Input{
		{Name: "основной", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: " ALPHA ", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: " ", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: "new", OpenedOn: "2026-02-30", CurrencyCode: "USD"},
		{Name: "new", OpenedOn: "2026-01-01", CurrencyCode: ""},
		{Name: "new", OpenedOn: "2026-01-01", CurrencyCode: "XXX"},
		{Name: "new", OpenedOn: "2026-01-01", CurrencyCode: "usd"},
	} {
		if _, err := s.Create(ctx, input); err == nil {
			t.Fatalf("accepted %+v", input)
		}
	}
	for _, input := range []portfolio.Input{
		{Name: "changed", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: "Alpha", OpenedOn: "2026-01-01", CurrencyCode: "RUB"},
		{Name: "changed", OpenedOn: "bad", CurrencyCode: "RUB"},
	} {
		if _, err := s.Update(ctx, a.ID, input); err == nil {
			t.Fatalf("accepted edit %+v", input)
		}
	}
	if _, err := s.Update(ctx, "99999", portfolio.Input{Name: "new", OpenedOn: "2026-01-01", CurrencyCode: "USD"}); !errors.Is(err, portfolio.ErrNotFound) {
		t.Fatal(err)
	}
	// Database constraints protect against bypassing the application and racing writes.
	for _, p := range []portfolio.Portfolio{
		{Name: "duplicate", NameKey: "alpha", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: "orphan", NameKey: "orphan", OpenedOn: "2026-01-01", CurrencyCode: "XXX"},
		{Name: "", NameKey: "empty", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: "  ", NameKey: "blank", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: "blank key", NameKey: "  ", OpenedOn: "2026-01-01", CurrencyCode: "USD"},
		{Name: "invalid date", NameKey: "invalid", OpenedOn: "2026-02-30", CurrencyCode: "USD"},
	} {
		if _, err := r.Create(ctx, p); err == nil {
			t.Fatalf("constraint accepted %+v", p)
		}
	}
	if _, err := store.DB().Exec("INSERT INTO portfolios(name, name_key, opened_on, currency_code) VALUES (NULL, 'null', '2026-01-01', 'USD')"); err == nil {
		t.Fatal("NULL name accepted")
	}
	collision := a
	collision.Name = "Alpha"
	collision.NameKey = "alpha"
	collision.OpenedOn = "2099-12-31"
	if _, err := r.Update(ctx, collision); err == nil {
		t.Fatal("duplicate update accepted")
	}
	after, _ := s.List(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed writes changed rows: %+v", after)
	}
	edited, err := s.Update(ctx, a.ID, portfolio.Input{Name: " Основной ", OpenedOn: "2100-01-01", CurrencyCode: "RUB"})
	if err != nil || edited.ID != a.ID || edited.Name != "Основной" {
		t.Fatalf("same name: %+v %v", edited, err)
	}
	edited, err = s.Update(ctx, a.ID, portfolio.Input{Name: "Новый", OpenedOn: "2100-02-28", CurrencyCode: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	store = open(t, path, migrations.Files)
	rows, err = portfolio.NewService(NewRepository(store.DB())).List(ctx)
	if err != nil || len(rows) != 3 || rows[0] != edited {
		t.Fatalf("restart: %+v %v", rows, err)
	}
}
func oldMigrations(t *testing.T) fstest.MapFS {
	t.Helper()
	result := fstest.MapFS{}
	for _, name := range []string{"0001_schema_migrations.sql", "0002_currency_directory.sql"} {
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		result[name] = &fstest.MapFile{Data: data}
	}
	return result
}
func TestMigrationBackupAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "rollback"}[fail], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db.sqlite3")
			old := oldMigrations(t)
			store := open(t, path, old)
			if _, err := store.DB().Exec("INSERT INTO currencies VALUES ('EUR')"); err != nil {
				t.Fatal(err)
			}
			if fail {
				if _, err := store.DB().Exec("CREATE TRIGGER reject_version BEFORE INSERT ON schema_migrations BEGIN SELECT RAISE(ABORT, 'test'); END"); err != nil {
					t.Fatal(err)
				}
			}
			_ = store.Close()
			updated, err := storage.Open(ctx, path, migrations.Files)
			if fail {
				if err == nil {
					updated.Close()
					t.Fatal("failed migration accepted")
				}
				updated = open(t, path, old)
			} else {
				if err != nil {
					t.Fatal(err)
				}
				defer updated.Close()
			}
			var tables int
			if err := updated.DB().QueryRow("SELECT count(*) FROM sqlite_master WHERE name = 'portfolios'").Scan(&tables); err != nil {
				t.Fatal(err)
			}
			if (tables == 1) == fail {
				t.Fatalf("portfolio table count %d", tables)
			}
			copies, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
			if len(copies) != 1 {
				t.Fatalf("backups: %v", copies)
			}
			backup := open(t, copies[0], old)
			var code string
			if err := backup.DB().QueryRow("SELECT code FROM currencies WHERE code='EUR'").Scan(&code); err != nil {
				t.Fatal("backup lost existing currency")
			}
			if !fail {
				rows, err := NewRepository(updated.DB()).List(ctx)
				if err != nil || len(rows) != 0 {
					t.Fatal("migration created portfolios")
				}
			}
		})
	}
}
func TestConcurrentDuplicateWrites(t *testing.T) {
	store := open(t, filepath.Join(t.TempDir(), "db.sqlite3"), migrations.Files)
	r := NewRepository(store.DB())
	p, _ := portfolio.Validate(portfolio.Input{Name: "Same", OpenedOn: "2026-01-01", CurrencyCode: "USD"})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := r.Create(ctx, p); results <- err })
	}
	wg.Wait()
	close(results)
	successes, duplicates := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			var e *portfolio.FieldError
			if errors.As(err, &e) && e.Field == "name" {
				duplicates++
			} else {
				t.Fatal(err)
			}
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("success=%d duplicates=%d", successes, duplicates)
	}
}
