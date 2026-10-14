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

func TestDepositSnapshotsCRUDAndIndependentRates(t *testing.T) {
	store, s, _ := setup(t)
	a, err := s.Save(ctx, "2", "", operations.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "94500.1234567890123456789", PortfolioAmount: "1000"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Save(ctx, "1", "", operations.Input{Kind: "deposit", Date: "2026-01-02", CurrencyCode: "USD", Amount: "10", PortfolioAmount: "900.2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", "", operations.Input{Kind: "withdrawal", Date: "2026-01-03", CurrencyCode: "RUB", Amount: "100", PortfolioAmount: "100"}); err != nil {
		t.Fatal(err)
	}
	v := valuation.NewService(valuationsqlite.NewRepository(store.DB()))
	out, err := v.Overview(ctx, "2026-01-04")
	if err != nil {
		t.Fatal(err)
	}
	if out.Points[3].Deposits != "95400.3234567890123456789" || !out.Points[3].DepositsComplete || out.Points[3].Complete {
		t.Fatal(out.Points)
	}
	if _, err = store.DB().Exec("INSERT INTO currency_rates VALUES('2026-01-01','USD','RUB','200')"); err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-04")
	if err != nil || out.Points[3].Deposits != "95400.3234567890123456789" || !out.Points[3].Complete {
		t.Fatal(out, err)
	}
	if _, err = store.DB().Exec("CREATE TRIGGER reject_deposit BEFORE UPDATE OF deposits_rub ON daily_cash_balances WHEN NEW.deposits_rub='999' BEGIN SELECT RAISE(ABORT,'test'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", b.ID, operations.Input{Kind: "deposit", Date: "2026-01-02", CurrencyCode: "RUB", Amount: "999", PortfolioAmount: "999"}); err == nil {
		t.Fatal("missing rollback")
	}
	if scalarString(t, store.DB(), "SELECT portfolio_amount FROM cash_flow_details WHERE operation_id=?", b.ID) != "900.2" {
		t.Fatal("partial write")
	}
	if _, err = s.Save(ctx, "1", b.ID, operations.Input{Kind: "deposit", Date: "2026-01-03", CurrencyCode: "RUB", Amount: "0.1", PortfolioAmount: "0.1"}); err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-04")
	if err != nil || out.Points[1].Deposits != "94500.1234567890123456789" || out.Points[3].Deposits != "94500.2234567890123456789" {
		t.Fatal(out, err)
	}
	if err = s.Delete(ctx, "2", a.ID); err != nil {
		t.Fatal(err)
	}
	missing, err := s.Save(ctx, "2", "", operations.Input{Kind: "deposit", Date: "2026-01-03", CurrencyCode: "USD", Amount: "10", PortfolioAmount: "10"})
	if err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-04")
	if err != nil || out.Points[1].DepositsComplete || out.Points[1].Deposits != "" {
		t.Fatal(out, err)
	}
	if err = s.Delete(ctx, "2", missing.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB().Exec("DROP TABLE cash_flow_details"); err != nil {
		t.Fatal(err)
	}
	out, err = v.Overview(ctx, "2026-01-04")
	if err != nil || out.Points[1].Deposits != "0.1" {
		t.Fatal(out, err)
	}
	for _, value := range []string{"-1", "1e3", "1.2.3", ".1", "1.", ""} {
		if _, err = store.DB().Exec("UPDATE daily_cash_balances SET deposits_rub=?", value); err == nil {
			t.Fatal("accepted", value)
		}
	}
}

func TestDepositMigrationAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "rollback"}[fail], func(t *testing.T) {
			old := fstest.MapFS{}
			entries, _ := fs.ReadDir(migrations.Files, ".")
			for _, e := range entries {
				if e.Name() < "0009" {
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
			_, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'P','p','2026-01-01','USD');INSERT INTO operations VALUES(1,1,'deposit','2026-01-01','');INSERT INTO cash_flow_details VALUES(1,'RUB','94500.1','1000');INSERT INTO daily_cash_balances VALUES(1,'2026-01-01','1000');`)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if _, err = store.DB().Exec("CREATE TRIGGER reject_9 BEFORE INSERT ON schema_migrations WHEN NEW.version=9 BEGIN SELECT RAISE(ABORT,'test'); END"); err != nil {
					t.Fatal(err)
				}
			}
			store.Close()
			updated, err := storage.Open(ctx, path, migrations.Files)
			if fail {
				if err == nil {
					updated.Close()
					t.Fatal("missing rollback")
				}
				updated, err = storage.Open(ctx, path, old)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer updated.Close()
			if fail {
				if scalarString(t, updated.DB(), "SELECT count(*) FROM pragma_table_info('daily_cash_balances') WHERE name='deposits_rub'") != "0" {
					t.Fatal("partial schema")
				}
			} else {
				if scalarString(t, updated.DB(), "SELECT deposits_rub FROM daily_cash_balances") != "94500.1" {
					t.Fatal("wrong backfill")
				}
			}
			copies, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
			if len(copies) != 1 {
				t.Fatal(copies)
			}
		})
	}
}
