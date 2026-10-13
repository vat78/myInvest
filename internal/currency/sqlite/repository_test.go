package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"myinvest/internal/currency"
	storage "myinvest/internal/storage/sqlite"
	shipped "myinvest/migrations"
)

func TestCurrencyDirectoryLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "currency.sqlite3")
	store := openTest(t, path, shipped.Files)
	s := currency.NewService(store)
	codes, err := s.List(ctx)
	if err != nil || len(codes) != 2 || codes[0] != "RUB" || codes[1] != "USD" {
		t.Fatalf("initial directory: %v %v", codes, err)
	}
	if v, found, err := s.Find(ctx, "2026-10-12", "USD", "RUB"); err != nil || found || v != "" {
		t.Fatalf("initial rate: %s %v %v", v, found, err)
	}
	for _, code := range []string{" eur ", "EUR"} {
		if got, err := s.Add(ctx, code); err != nil || got != "EUR" {
			t.Fatalf("add: %q %v", got, err)
		}
	}
	if n := scalar(t, store.db, "SELECT count(*) FROM currencies"); n != 3 {
		t.Fatalf("duplicate currencies: %d", n)
	}
	long := "95.123456789012345678901234567890123456789000"
	for _, row := range []struct{ date, base, quote, value string }{
		{"2026-10-12", "USD", "RUB", "95,125"}, {"2026-10-12", "USD", "RUB", long},
		{"2026-10-11", "USD", "RUB", "94"},
		{"2026-10-12", "EUR", "RUB", "104.50"}, {"2026-10-12", "EUR", "RUB", "105.50"},
	} {
		if err := s.Save(ctx, row.date, row.base, row.quote, row.value); err != nil {
			t.Fatal(err)
		}
	}
	if n := scalar(t, store.db, "SELECT count(*) FROM currency_rates"); n != 6 {
		t.Fatalf("upsert rows: %d", n)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTest(t, path, shipped.Files)
	s = currency.NewService(store)
	if got, found, err := s.Find(ctx, "2026-10-12", " usd ", "rUb"); err != nil || !found || got != long {
		t.Fatalf("reopened exact rate: %q %v %v", got, found, err)
	}
	if got, found, err := s.Find(ctx, "2026-10-12", "EUR", "RUB"); err != nil || !found || got != "105.50" {
		t.Fatalf("non-USD pair: %q %v %v", got, found, err)
	}
	assertRate(t, s, "2026-10-12", "RUB", "EUR", "0.009478672985781991")
	assertRate(t, s, "2026-10-12", "RUB", "USD", "0.010512654120823639")
	if _, found, err := s.Find(ctx, "2026-10-13", "USD", "RUB"); err != nil || found {
		t.Fatal("previous day carried forward")
	}
	if _, found, err := s.Find(ctx, "2026-10-12", "USD", "EUR"); err != nil || found {
		t.Fatal("implicit cross-rate")
	}
	var kind string
	if err := store.db.QueryRow("SELECT typeof(value) FROM currency_rates LIMIT 1").Scan(&kind); err != nil || kind != "text" {
		t.Fatalf("storage type: %q %v", kind, err)
	}
}

func TestCurrencyInvalidWritesAreAtomic(t *testing.T) {
	store := openTest(t, filepath.Join(t.TempDir(), "currency.sqlite3"), shipped.Files)
	s := currency.NewService(store)
	if err := s.Save(ctx, "2026-10-12", "USD", "RUB", "95"); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ date, base, quote, value string }{
		{"2026-02-30", "USD", "RUB", "96"}, {"2026-10-12", "USD", "USD", "96"},
		{"2026-10-12", "XXX", "RUB", "96"}, {"2026-10-12", "BAD!", "RUB", "96"},
		{"2026-10-12", "USD", "RUB", "0"}, {"2026-10-12", "USD", "RUB", "-1"},
		{"2026-10-12", "USD", "RUB", "4000000000000000000"},
		{"2026-10-12", "USD", "RUB", "1e4"}, {"2026-10-12", "USD", "RUB", ""},
	} {
		if err := s.Save(ctx, row.date, row.base, row.quote, row.value); err == nil {
			t.Fatalf("accepted %+v", row)
		}
	}
	if err := store.SaveRates(ctx, currency.Rate{Pair: currency.Pair{Date: "2026-10-12", Base: "XXX", Quote: "RUB"}, Value: "1"}, currency.Rate{Pair: currency.Pair{Date: "2026-10-12", Base: "RUB", Quote: "XXX"}, Value: "1"}); err == nil {
		t.Fatal("orphan accepted")
	}
	for _, code := range []string{"U1D", "USD; DROP TABLE currencies", "РУБ"} {
		if _, err := s.Add(ctx, code); err == nil {
			t.Fatal("invalid code accepted")
		}
		if err := store.AddCurrency(ctx, code); err == nil {
			t.Fatal("SQL code constraint failed")
		}
	}
	for _, row := range []struct{ date, base, quote, value string }{
		{"2026-02-30", "USD", "RUB", "1"}, {"bad", "USD", "RUB", "1"},
		{"2026-10-12", "USD", "USD", "1"}, {"2026-10-12", "USD", "RUB", "0.000"},
		{"2026-10-12", "USD", "RUB", "-1"}, {"2026-10-12", "USD", "RUB", "1..2"},
		{"2026-10-12", "USD", "RUB", ".5"}, {"2026-10-12", "USD", "RUB", "5."},
	} {
		if _, err := store.db.Exec("INSERT OR REPLACE INTO currency_rates VALUES (?, ?, ?, ?)", row.date, row.base, row.quote, row.value); err == nil {
			t.Fatalf("SQL constraint accepted %+v", row)
		}
	}
	if n := scalar(t, store.db, "SELECT count(*) FROM currency_rates"); n != 2 {
		t.Fatal("partial rate write")
	}
	if n := scalar(t, store.db, "SELECT count(*) FROM currencies"); n != 2 {
		t.Fatal("implicit currency creation")
	}
	if got, _, _ := s.Find(ctx, "2026-10-12", "USD", "RUB"); got != "95" {
		t.Fatal("failed save changed existing rate")
	}
}

func TestCurrencyMigrationUpgradeAndRollback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "currency.sqlite3")
	initial := openTest(t, path, scripts(t, nil))
	_ = initial.Close()
	updated := openTest(t, path, shipped.Files)
	if n := scalar(t, updated.db, "SELECT count(*) FROM schema_migrations"); n != 3 {
		t.Fatal("migration missing")
	}
	copies, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if len(copies) != 1 {
		t.Fatalf("upgrade backup: %v", copies)
	}
	backup := openTest(t, copies[0], scripts(t, nil))
	if n := scalar(t, backup.db, "SELECT count(*) FROM sqlite_master WHERE name = 'currencies'"); n != 0 {
		t.Fatal("backup already migrated")
	}

	brokenPath := filepath.Join(t.TempDir(), "currency.sqlite3")
	broken := openTest(t, brokenPath, scripts(t, nil))
	execSQL(t, broken.db, "CREATE TRIGGER reject_version BEFORE INSERT ON schema_migrations BEGIN SELECT RAISE(ABORT, 'test'); END")
	_ = broken.Close()
	if _, err := storage.Open(ctx, brokenPath, shipped.Files); err == nil {
		t.Fatal("migration recording failure ignored")
	}
	reopened := openTest(t, brokenPath, scripts(t, nil))
	if n := scalar(t, reopened.db, "SELECT count(*) FROM sqlite_master WHERE name IN ('currencies', 'currency_rates')"); n != 0 {
		t.Fatal("currency migration did not roll back")
	}
}

var ctx = context.Background()

// Only test databases expose SQL for constraint and migration assertions.
type testDatabase struct {
	*Repository
	db    *sql.DB
	store *storage.Store
}

func (db *testDatabase) Close() error { return db.store.Close() }
func openTest(t *testing.T, path string, files fs.FS) *testDatabase {
	t.Helper()
	store, err := storage.Open(ctx, path, files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &testDatabase{Repository: NewRepository(store.DB()), db: store.DB(), store: store}
}
func scripts(t *testing.T, extra map[string]string) fstest.MapFS {
	t.Helper()
	bootstrap, err := fs.ReadFile(shipped.Files, "0001_schema_migrations.sql")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{"0001_schema_migrations.sql": &fstest.MapFile{Data: bootstrap}}
	for name, body := range extra {
		files[name] = &fstest.MapFile{Data: []byte(body)}
	}
	return files
}
func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
func scalar(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var value int
	if err := db.QueryRowContext(ctx, query).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func assertRate(t *testing.T, s *currency.Service, date, base, quote, want string) {
	t.Helper()
	value, found, err := s.Find(ctx, date, base, quote)
	if err != nil || !found || value != want {
		t.Fatalf("%s %s/%s: %q, found=%v, err=%v; want %q", date, base, quote, value, found, err, want)
	}
}

func TestInverseRatesUpdateAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inverse.sqlite3")
	store := openTest(t, path, shipped.Files)
	s := currency.NewService(store)
	if err := s.Save(ctx, "2026-10-13", "USD", "RUB", "95,125"); err != nil {
		t.Fatal(err)
	}
	assertRate(t, s, "2026-10-13", "USD", "RUB", "95.125")
	assertRate(t, s, "2026-10-13", "RUB", "USD", "0.010512483574244415")
	if err := s.Save(ctx, "2026-10-13", "USD", "RUB", "100.000"); err != nil {
		t.Fatal(err)
	}
	assertRate(t, s, "2026-10-13", "USD", "RUB", "100.000")
	assertRate(t, s, "2026-10-13", "RUB", "USD", "0.01")
	// Editing the opposite direction preserves that input and recomputes its inverse.
	if err := s.Save(ctx, "2026-10-13", "RUB", "USD", "0,0200"); err != nil {
		t.Fatal(err)
	}
	assertRate(t, s, "2026-10-13", "RUB", "USD", "0.0200")
	assertRate(t, s, "2026-10-13", "USD", "RUB", "50")
	if n := scalar(t, store.db, "SELECT count(*) FROM currency_rates"); n != 2 {
		t.Fatalf("duplicate rates: %d", n)
	}
	if err := s.Save(ctx, "2026-10-12", "USD", "RUB", "95"); err != nil {
		t.Fatal(err)
	}
	assertRate(t, s, "2026-10-13", "USD", "RUB", "50")
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTest(t, path, shipped.Files)
	s = currency.NewService(store)
	assertRate(t, s, "2026-10-13", "RUB", "USD", "0.0200")
	assertRate(t, s, "2026-10-13", "USD", "RUB", "50")
	assertRate(t, s, "2026-10-12", "RUB", "USD", "0.010526315789473684")
}

func TestInverseFailureRollsBackBothDirections(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			store := openTest(t, filepath.Join(t.TempDir(), "inverse.sqlite3"), shipped.Files)
			s := currency.NewService(store)
			if existing {
				if err := s.Save(ctx, "2026-10-13", "USD", "RUB", "95"); err != nil {
					t.Fatal(err)
				}
			}
			// Fail the second write after the first direction has been inserted/updated.
			execSQL(t, store.db, `CREATE TRIGGER reject_inverse BEFORE INSERT ON currency_rates
				WHEN NEW.base = 'RUB' BEGIN SELECT RAISE(ABORT, 'inverse unavailable'); END`)
			if err := s.Save(ctx, "2026-10-13", "USD", "RUB", "100"); err == nil {
				t.Fatal("second write failure ignored")
			}
			if existing {
				assertRate(t, s, "2026-10-13", "USD", "RUB", "95")
				assertRate(t, s, "2026-10-13", "RUB", "USD", "0.010526315789473684")
			} else if n := scalar(t, store.db, "SELECT count(*) FROM currency_rates"); n != 0 {
				t.Fatal("partial insert persisted")
			}
			execSQL(t, store.db, "DROP TRIGGER reject_inverse")
			if err := s.Save(ctx, "2026-10-13", "USD", "RUB", "100"); err != nil {
				t.Fatal(err)
			}
			assertRate(t, s, "2026-10-13", "RUB", "USD", "0.01")
		})
	}
}

func TestExistingRatesRemainUntilExplicitSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite3")
	store := openTest(t, path, shipped.Files)
	execSQL(t, store.db, "INSERT INTO currency_rates VALUES ('2026-10-12', 'USD', 'RUB', '95')")
	execSQL(t, store.db, "INSERT INTO currency_rates VALUES ('2026-10-12', 'RUB', 'USD', '0.02')")
	_ = store.Close()
	store = openTest(t, path, shipped.Files)
	s := currency.NewService(store)
	assertRate(t, s, "2026-10-12", "USD", "RUB", "95")
	assertRate(t, s, "2026-10-12", "RUB", "USD", "0.02")
	if err := s.Save(ctx, "2026-10-12", "USD", "RUB", "95"); err != nil {
		t.Fatal(err)
	}
	assertRate(t, s, "2026-10-12", "RUB", "USD", "0.010526315789473684")
}
