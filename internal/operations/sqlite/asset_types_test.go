package sqlite_test

import (
	"errors"
	"io/fs"
	"myinvest/internal/operations"
	. "myinvest/internal/operations/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestAssetTypesPersistenceValidationAndRollback(t *testing.T) {
	store, s, path := setup(t)
	for _, v := range operations.AssetTypes() {
		a, err := s.AddAsset(ctx, v.Code, v.Code)
		if err != nil || a.TypeCode != v.Code {
			t.Fatal(a, err)
		}
	}
	before, _ := s.ListAssets(ctx)
	for _, code := range []string{"", "invalid"} {
		if _, err := s.AddAsset(ctx, "NEW", code); err == nil {
			t.Fatal("invalid add", code)
		}
		if _, err := s.UpdateAssetType(ctx, before[0].ID, code); err == nil {
			t.Fatal("invalid update", code)
		}
		for _, query := range []string{"INSERT INTO assets(ticker,type_code) VALUES ('NEW',?)", "UPDATE assets SET type_code=? WHERE id=1"} {
			if _, err := store.DB().Exec(query, code); err == nil {
				t.Fatal("missing CHECK", code)
			}
		}
	}
	if _, err := s.AddAsset(ctx, "  "+before[0].Ticker+"  ", "stocks"); err == nil {
		t.Fatal("duplicate")
	}
	if _, err := s.UpdateAssetType(ctx, "9999", "bonds"); !errors.Is(err, operations.ErrAssetNotFound) {
		t.Fatal(err)
	}
	after, _ := s.ListAssets(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed writes changed assets")
	}
	a, err := s.UpdateAssetType(ctx, before[0].ID, "bonds")
	if err != nil || a.ID != before[0].ID || a.Ticker != before[0].Ticker || a.TypeCode != "bonds" {
		t.Fatal(a, err)
	}
	if _, err := store.DB().Exec(`CREATE TRIGGER reject_type BEFORE UPDATE OF type_code ON assets BEGIN SELECT RAISE(ABORT,'private SQL failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateAssetType(ctx, a.ID, "stocks"); err == nil {
		t.Fatal("failure committed")
	}
	if got := scalarString(t, store.DB(), "SELECT type_code FROM assets WHERE id=?", a.ID); got != "bonds" {
		t.Fatal(got)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s = operations.NewService(NewRepository(store.DB()))
	rows, err := s.ListAssets(ctx)
	if err != nil || len(rows) != 7 || rows[0] != a {
		t.Fatal(rows, err)
	}
}

func TestAssetTypeMigrationPreservesHistory(t *testing.T) {
	old := legacyFiles(t)
	data, err := fs.ReadFile(migrations.Files, "0005_unified_operations.sql")
	if err != nil {
		t.Fatal(err)
	}
	old["0005_unified_operations.sql"] = &fstest.MapFile{Data: data}
	path := filepath.Join(t.TempDir(), "legacy.sqlite3")
	store, err := storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'Legacy','legacy','2026-01-01','RUB');
 INSERT INTO assets(id,ticker) VALUES(42,'OLD');
 INSERT INTO operations(id,portfolio_id,kind,date,comment) VALUES(7,1,'buy','2026-01-01','legacy');
 INSERT INTO trade_details VALUES(7,42,'2.5','10.01','0.1','4.004');
 INSERT INTO daily_cash_balances VALUES(1,'2026-01-01','-10.11');
 INSERT INTO daily_asset_balances VALUES(1,'2026-01-01',42,'2.5');`)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := operations.NewService(NewRepository(store.DB()))
	rows, err := s.ListAssets(ctx)
	if err != nil || len(rows) != 1 || rows[0] != (operations.Asset{ID: "42", Ticker: "OLD"}) {
		t.Fatal(rows, err)
	}
	if scalarString(t, store.DB(), "SELECT type_code IS NULL FROM assets WHERE id=42") != "1" {
		t.Fatal("legacy classified")
	}
	before, err := s.List(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	// Legacy unclassified assets remain tradable.
	op, err := s.Save(ctx, "1", "", trade("sell", "42", "0.5", "3", "0", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "1", op.ID); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"stocks", "bonds"} {
		a, err := s.UpdateAssetType(ctx, "42", code)
		if err != nil || a.ID != "42" || a.Ticker != "OLD" || a.TypeCode != code {
			t.Fatal(a, err)
		}
		after, err := s.List(ctx, "1")
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal(after, err)
		}
		if scalarString(t, store.DB(), "SELECT balance FROM daily_cash_balances") != "-10.11" || scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances") != "2.5" {
			t.Fatal("totals changed")
		}
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
	if scalarString(t, backup.DB(), "SELECT ticker FROM assets WHERE id=42") != "OLD" {
		t.Fatal("backup")
	}
}

func TestAssetTypeMigrationRollback(t *testing.T) {
	old := legacyFiles(t)
	data, err := fs.ReadFile(migrations.Files, "0005_unified_operations.sql")
	if err != nil {
		t.Fatal(err)
	}
	old["0005_unified_operations.sql"] = &fstest.MapFile{Data: data}
	path := filepath.Join(t.TempDir(), "legacy.sqlite3")
	store, err := storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB().Exec(`INSERT INTO assets(id,ticker) VALUES(42,'OLD');
 CREATE TRIGGER reject_6 BEFORE INSERT ON schema_migrations WHEN NEW.version=6 BEGIN SELECT RAISE(ABORT,'test'); END;`)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	if reopened, err := storage.Open(ctx, path, migrations.Files); err == nil {
		reopened.Close()
		t.Fatal("failed migration committed")
	}
	store, err = storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if scalarString(t, store.DB(), "SELECT ticker FROM assets WHERE id=42") != "OLD" {
		t.Fatal("legacy lost")
	}
	if scalarString(t, store.DB(), "SELECT count(*) FROM pragma_table_info('assets') WHERE name='type_code'") != "0" {
		t.Fatal("column committed")
	}
}
