package sqlite_test

import (
	"io/fs"
	"myinvest/internal/operations"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/internal/valuation"
	valuationsqlite "myinvest/internal/valuation/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDailyValuesCRUDAndReadWithoutJournal(t *testing.T) {
	store, s, _ := setup(t)
	asset, err := s.AddAsset(ctx, "A", "stocks")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(ctx, "1", "", operations.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1000", PortfolioAmount: "1000"})
	if err != nil {
		t.Fatal(err)
	}
	buy, err := s.Save(ctx, "1", "", trade("buy", asset.ID, "2.5", "250", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		"UPDATE daily_asset_balances SET value_denominator='0'",
		"UPDATE daily_asset_balances SET value_denominator=NULL",
		"UPDATE daily_asset_balances SET value_numerator='1e3'",
		"UPDATE daily_asset_balances SET price_operation_id=999",
	} {
		if _, err = store.DB().Exec(query); err == nil {
			t.Fatal("accepted invalid valuation", query)
		}
	}
	v := valuation.NewService(valuationsqlite.NewRepository(store.DB()))
	out, err := v.Overview(ctx, "2026-01-02")
	if err != nil || out.Total.Value != "1000" {
		t.Fatal(out, err)
	}
	if scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances") != "250" {
		t.Fatal("missing stored value")
	}
	split, err := s.Save(ctx, "1", "", operations.Input{Kind: "split", Date: "2026-01-02", AssetID: asset.ID, OldUnits: "1", NewUnits: "2"})
	if err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-02")
	if err != nil || out.Total.Value != "1000" {
		t.Fatal(out, err)
	}
	if _, err = s.Save(ctx, "1", buy.ID, trade("buy", asset.ID, "2.5", "300", "0", "2026-01-01")); err != nil {
		t.Fatal(err)
	}
	if scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances WHERE date='2026-01-02'") != "300" {
		t.Fatal("stale history")
	}
	if _, err = store.DB().Exec("CREATE TRIGGER reject_value BEFORE INSERT ON daily_asset_balances WHEN NEW.value_numerator='350' BEGIN SELECT RAISE(ABORT,'test'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", buy.ID, trade("buy", asset.ID, "2.5", "350", "0", "2026-01-01")); err == nil {
		t.Fatal("missing rollback")
	}
	if scalarString(t, store.DB(), "SELECT amount FROM trade_details WHERE operation_id=?", buy.ID) != "300" {
		t.Fatal("partial write")
	}
	if err = s.Delete(ctx, "1", split.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, "1", buy.ID); err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-02")
	if err != nil || out.Total.Value != "1000" {
		t.Fatal(out, err)
	}
	// Persisted snapshots suffice even if the journal cannot be read.
	if _, err = store.DB().Exec("DROP TABLE trade_details"); err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-02")
	if err != nil || out.Total.Value != "1000" {
		t.Fatal(out, err)
	}
}
func TestValuationMigrationAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "rollback"}[fail], func(t *testing.T) {
			old := fstest.MapFS{}
			entries, _ := fs.ReadDir(migrations.Files, ".")
			for _, e := range entries {
				if e.Name() < "0008" {
					data, err := fs.ReadFile(migrations.Files, e.Name())
					if err != nil {
						t.Fatal(err)
					}
					old[e.Name()] = &fstest.MapFile{Data: data}
				}
			}
			path := filepath.Join(t.TempDir(), "db.sqlite3")
			store, err := storage.Open(ctx, path, old)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'P','p','2026-01-01','RUB');INSERT INTO assets(id,ticker,type_code) VALUES(1,'A','stocks');INSERT INTO operations VALUES(1,1,'buy','2026-01-01','');INSERT INTO trade_details VALUES(1,1,'3','1','0','0.333333333333333333');INSERT INTO daily_cash_balances VALUES(1,'2026-01-01','-1');INSERT INTO daily_asset_balances VALUES(1,'2026-01-01',1,'3');`)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if _, err = store.DB().Exec("CREATE TRIGGER reject_8 BEFORE INSERT ON schema_migrations WHEN NEW.version=8 BEGIN SELECT RAISE(ABORT,'test'); END"); err != nil {
					t.Fatal(err)
				}
			}
			store.Close()
			updated, err := storage.Open(ctx, path, migrations.Files)
			if fail {
				if err == nil {
					updated.Close()
					t.Fatal("migration succeeded")
				}
				updated, err = storage.Open(ctx, path, old)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer updated.Close()
			if fail {
				if scalarString(t, updated.DB(), "SELECT count(*) FROM pragma_table_info('daily_asset_balances') WHERE name='value_numerator'") != "0" {
					t.Fatal("partial schema")
				}
			} else {
				if scalarString(t, updated.DB(), "SELECT value_numerator FROM daily_asset_balances") != "999999999999999999" || scalarString(t, updated.DB(), "SELECT value_denominator FROM daily_asset_balances") != "1000000000000000000" {
					t.Fatal("incorrect migration value")
				}
			}
			copies, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
			if len(copies) != 1 {
				t.Fatal(copies)
			}
		})
	}
}
