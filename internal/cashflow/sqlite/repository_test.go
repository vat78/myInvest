package sqlite

import (
	"context"
	"errors"
	"io/fs"
	"myinvest/internal/cashflow"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"testing"
	"testing/fstest"
)

var ctx = context.Background()

func setup(t *testing.T) (*storage.Store, *cashflow.Service, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	store, err := storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if _, err = store.DB().Exec("INSERT INTO portfolios(id,name,name_key,opened_on,currency_code) VALUES (1,'Первый','первый','2026-01-01','RUB'),(2,'Второй','второй','2026-01-01','USD')"); err != nil {
		t.Fatal(err)
	}
	return store, cashflow.NewService(NewRepository(store.DB())), path
}
func input(kind, a, b string) cashflow.Input {
	return cashflow.Input{Kind: kind, Date: "2026-01-01", CurrencyCode: "USD", Amount: a, PortfolioAmount: b, Comment: "С удержаниями\nточно"}
}
func TestLifecycleAndIndependence(t *testing.T) {
	store, s, path := setup(t)
	initial, err := s.List(ctx, "1")
	if err != nil || initial.Balance != "0" || len(initial.Operations) != 0 {
		t.Fatal(initial, err)
	}
	d, err := s.Save(ctx, "1", "", input("deposit", "1000", "94500"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.Save(ctx, "1", "", input("withdrawal", "500", "48000"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.List(ctx, "1")
	if err != nil || snapshot.Balance != "46500" || snapshot.Operations[0].ID != w.ID {
		t.Fatal(snapshot, err)
	}
	if _, err = store.DB().Exec("INSERT INTO currency_rates(date,base,quote,value) VALUES ('2026-01-01','USD','RUB','123')"); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	s = cashflow.NewService(NewRepository(reopened.DB()))
	snapshot, err = s.List(ctx, "1")
	if err != nil || snapshot.Balance != "46500" || snapshot.Operations[1].Comment != d.Comment || snapshot.Operations[1].Date != d.Date || snapshot.Operations[0].Amount != "500" {
		t.Fatal(snapshot, err)
	}
	if _, err = reopened.DB().Exec("UPDATE currency_rates SET value='99'"); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = s.List(ctx, "1")
	if snapshot.Balance != "46500" {
		t.Fatal(snapshot)
	}
	other, _ := s.List(ctx, "2")
	if other.Balance != "0" || len(other.Operations) != 0 {
		t.Fatal(other)
	}
	if _, err = s.Save(ctx, "2", d.ID, input("deposit", "1", "1")); !errors.Is(err, cashflow.ErrNotFound) {
		t.Fatal("moved operation", err)
	}
	if err = s.Delete(ctx, "2", d.ID); !errors.Is(err, cashflow.ErrNotFound) {
		t.Fatal("deleted other portfolio", err)
	}
	edited, err := s.Save(ctx, "1", d.ID, input("deposit", "2", "0.1"))
	if err != nil || edited.ID != d.ID || edited.PortfolioID != "1" {
		t.Fatal(edited, err)
	}
	snapshot, _ = s.List(ctx, "1")
	if snapshot.Balance != "-47999.9" {
		t.Fatal(snapshot)
	}
	if err = s.Delete(ctx, "1", w.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = s.List(ctx, "1")
	if snapshot.Balance != "0.1" {
		t.Fatal(snapshot)
	}
	if err = s.Delete(ctx, "1", d.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = s.List(ctx, "1")
	if snapshot.Balance != "0" {
		t.Fatal(snapshot)
	}
	newer, err := s.Save(ctx, "1", "", input("deposit", "1", "1"))
	if err != nil || newer.ID == d.ID || newer.ID == w.ID {
		t.Fatal("reused ID", newer, err)
	}
}
func TestInvalidAndFailedWrites(t *testing.T) {
	store, s, _ := setup(t)
	valid, err := s.Save(ctx, "1", "", input("deposit", "1", "0.123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*cashflow.Input){func(in *cashflow.Input) { in.Kind = "other" }, func(in *cashflow.Input) { in.Date = "2026-02-30" }, func(in *cashflow.Input) { in.CurrencyCode = "EUR" }, func(in *cashflow.Input) { in.Amount = "-1" }, func(in *cashflow.Input) { in.PortfolioAmount = "0" }, func(in *cashflow.Input) { in.CurrencyCode = "RUB" }} {
		in := input("deposit", "1", "2")
		change(&in)
		if _, err = s.Save(ctx, "1", valid.ID, in); err == nil {
			t.Fatal("accepted invalid", in)
		}
	}
	if _, err = s.Save(ctx, "99", "", input("deposit", "1", "2")); !errors.Is(err, cashflow.ErrPortfolio) {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", "99", input("deposit", "1", "2")); !errors.Is(err, cashflow.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = store.DB().Exec("CREATE TRIGGER reject_write BEFORE UPDATE ON cash_flows BEGIN SELECT RAISE(ABORT,'private SQL details'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", valid.ID, input("withdrawal", "1", "99")); err == nil {
		t.Fatal("failed write succeeded")
	}
	if _, err = store.DB().Exec("CREATE TRIGGER reject_delete BEFORE DELETE ON cash_flows BEGIN SELECT RAISE(ABORT,'private SQL details'); END"); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, "1", valid.ID); err == nil {
		t.Fatal("failed delete succeeded")
	}
	snapshot, err := s.List(ctx, "1")
	if err != nil || snapshot.Balance != valid.PortfolioAmount || len(snapshot.Operations) != 1 {
		t.Fatal(snapshot, err)
	}
	// A failure after a successful write must roll back the whole transaction.
	r := NewRepository(store.DB())
	err = r.Transaction(ctx, func(tx cashflow.Repository) error {
		if _, err := tx.Save(ctx, cashflow.Operation{PortfolioID: "1", Input: input("deposit", "3", "4")}); err != nil {
			return err
		}
		return errors.New("abort")
	})
	if err == nil {
		t.Fatal("no failure")
	}
	snapshot, _ = s.List(ctx, "1")
	if len(snapshot.Operations) != 1 {
		t.Fatal("partial transaction", snapshot)
	}
}
func TestConstraints(t *testing.T) {
	store, _, _ := setup(t)
	for _, args := range [][]any{{99, "deposit", "2026-01-01", "USD", "1", "1"}, {1, "deposit", "2026-01-01", "EUR", "1", "1"}, {1, "trade", "2026-01-01", "USD", "1", "1"}, {1, "deposit", "2026-02-30", "USD", "1", "1"}, {1, "deposit", "2026-01-01", "USD", "0", "1"}, {1, "deposit", "2026-01-01", "USD", "1", "-1"}, {1, "deposit", "2026-01-01", "USD", "1e2", "1"}, {1, "deposit", "2026-01-01", "USD", nil, "1"}} {
		if _, err := store.DB().Exec("INSERT INTO cash_flows(portfolio_id,kind,date,currency_code,amount,portfolio_amount,comment) VALUES (?,?,?,?,?,?,'')", args...); err == nil {
			t.Fatal("accepted invalid row", args)
		}
	}
}
func TestMigrationUpgradeAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "rollback"}[fail], func(t *testing.T) {
			old := fstest.MapFS{}
			for _, name := range []string{"0001_schema_migrations.sql", "0002_currency_directory.sql", "0003_portfolios.sql"} {
				data, err := fs.ReadFile(migrations.Files, name)
				if err != nil {
					t.Fatal(err)
				}
				old[name] = &fstest.MapFile{Data: data}
			}
			path := filepath.Join(t.TempDir(), "db.sqlite3")
			store, err := storage.Open(ctx, path, old)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.DB().Exec("INSERT INTO portfolios VALUES (1,'Existing','existing','2026-01-01','RUB')")
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				_, err = store.DB().Exec("CREATE TRIGGER reject_version BEFORE INSERT ON schema_migrations BEGIN SELECT RAISE(ABORT,'test'); END")
				if err != nil {
					t.Fatal(err)
				}
			}
			store.Close()
			updated, err := storage.Open(ctx, path, migrations.Files)
			if fail {
				if err == nil {
					updated.Close()
					t.Fatal("accepted failed migration")
				}
				updated, err = storage.Open(ctx, path, old)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer updated.Close()
			var count int
			if err = updated.DB().QueryRow("SELECT count(*) FROM sqlite_master WHERE name='cash_flows'").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if (count == 1) == fail {
				t.Fatal(count)
			}
			var name string
			if err = updated.DB().QueryRow("SELECT name FROM portfolios WHERE id=1").Scan(&name); err != nil || name != "Existing" {
				t.Fatal(name, err)
			}
			copies, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
			if len(copies) != 1 {
				t.Fatal(copies)
			}
			backup, err := storage.Open(ctx, copies[0], old)
			if err != nil {
				t.Fatal(err)
			}
			defer backup.Close()
			if err = backup.DB().QueryRow("SELECT name FROM portfolios WHERE id=1").Scan(&name); err != nil || name != "Existing" {
				t.Fatal(name, err)
			}
			if !fail {
				rows, err := NewRepository(updated.DB()).List(ctx, "1")
				if err != nil || len(rows) != 0 {
					t.Fatal(rows, err)
				}
			}
		})
	}
}
