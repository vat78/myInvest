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
	"sync"
	"testing"
	"testing/fstest"
)

func split(a, old, next, date string) operations.Input {
	return operations.Input{Kind: "split", AssetID: a, OldUnits: old, NewUnits: next, Date: date, Comment: "split comment"}
}
func TestSplitLifecycleAndPreview(t *testing.T) {
	store, s, path := setup(t)
	a, err := s.AddAsset(ctx, "SBER", "stocks")
	if err != nil {
		t.Fatal(err)
	}
	buy, err := s.Save(ctx, "1", "", trade("buy", a.ID, "2", "20", "1", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "2", "", trade("buy", a.ID, "12", "1", "0", "2026-01-01")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", "", trade("buy", a.ID, "3", "30", "0", "2026-01-02")); err != nil {
		t.Fatal(err)
	}
	sale, err := s.Save(ctx, "1", "", trade("sell", a.ID, "5", "50", "2", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewSplit(ctx, "1", "", split(a.ID, "1", "10", "2026-01-02"))
	if err != nil || preview != (operations.SplitQuantities{Before: "2", After: "20"}) {
		t.Fatal(preview, err)
	}
	if scalarString(t, store.DB(), "SELECT count(*) FROM operations") != "4" {
		t.Fatal("preview wrote rows")
	}
	op, err := s.Save(ctx, "1", "", split(a.ID, "1", "10", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.List(ctx, "1")
	if err != nil || snap.Balance != "-3" || len(snap.Warnings) != 0 || snap.Operations[0].ID != op.ID || snap.Operations[1].ID != sale.ID || snap.Operations[0].QuantityBefore != "2" || snap.Operations[0].QuantityAfter != "20" {
		t.Fatal(snap, err)
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-02'") != "18" {
		t.Fatal("wrong daily quantity")
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE portfolio_id=2") != "12" {
		t.Fatal("other portfolio changed")
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM trade_details WHERE operation_id=?", buy.ID) != "2" {
		t.Fatal("original trade changed")
	}
	// Edit date and ratio: same-day trades are excluded, old date removed from split application.
	op, err = s.Save(ctx, "1", op.ID, split(a.ID, "10", "1", "2026-01-03"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || len(snap.Warnings) != 0 || snap.Operations[0].QuantityAfter != "0" {
		t.Fatal(snap, err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	s = operations.NewService(NewRepository(reopened.DB()))
	again, err := s.List(ctx, "1")
	if err != nil || !reflect.DeepEqual(snap, again) {
		t.Fatal(again, err)
	}
	if err = s.Delete(ctx, "1", op.ID); err != nil {
		t.Fatal(err)
	}
	if scalarString(t, reopened.DB(), "SELECT count(*) FROM split_details") != "0" || scalarString(t, reopened.DB(), "SELECT count(*) FROM daily_cash_balances WHERE portfolio_id=1 AND date='2026-01-03'") != "0" {
		t.Fatal("split deletion left data")
	}
}
func TestSplitHistoricalRollback(t *testing.T) {
	store, s, _ := setup(t)
	a, _ := s.AddAsset(ctx, "SBER", "stocks")
	buy, err := s.Save(ctx, "1", "", trade("buy", a.ID, "1", "1", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	early, err := s.Save(ctx, "1", "", split(a.ID, "1", "3", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	late, err := s.Save(ctx, "1", "", split(a.ID, "3", "1", "2026-01-03"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.List(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewSplit(ctx, "1", early.ID, split(a.ID, "1", "2", "2026-01-02")); err == nil {
		t.Fatal("preview ignored downstream periodic split")
	}
	for _, mutate := range []func() error{
		func() error {
			_, e := s.Save(ctx, "1", buy.ID, trade("buy", a.ID, "0.1", "1", "0", "2026-01-02"))
			return e
		},
		func() error { _, e := s.Save(ctx, "1", early.ID, split(a.ID, "1", "2", "2026-01-02")); return e },
		func() error { return s.Delete(ctx, "1", early.ID) },
		func() error { _, e := s.Save(ctx, "1", "", trade("buy", a.ID, "1", "1", "0", "2026-01-02")); return e },
		func() error { _, e := s.Save(ctx, "1", "", split(a.ID, "3", "1", "2026-01-02")); return e },
	} {
		err = mutate()
		var e *operations.SplitError
		if !errors.As(err, &e) || e.Ticker != "SBER" {
			t.Fatal(err)
		}
		after, e2 := s.List(ctx, "1")
		if e2 != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("partial history", after, e2)
		}
		if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-03'") != "1" {
			t.Fatal("partial daily balances")
		}
	}
	// Deleting an early purchase can also invalidate a later split.
	extra, err := s.Save(ctx, "1", "", trade("buy", a.ID, "3", "1", "0", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", buy.ID, trade("buy", a.ID, "1", "1", "0", "2026-01-02")); err == nil {
		t.Fatal("periodic move accepted")
	}
	if err = s.Delete(ctx, "1", buy.ID); err != nil {
		t.Fatal(err)
	} // Remaining 3 / 3 is finite.
	_, err = s.Save(ctx, "1", extra.ID, trade("buy", a.ID, "1", "1", "0", "2026-01-02"))
	if err == nil {
		t.Fatal("periodic history accepted")
	}
	if err = s.Delete(ctx, "1", late.ID); err != nil {
		t.Fatal(err)
	}
}
func TestSplitDeletePurchaseRollback(t *testing.T) {
	store, s, _ := setup(t)
	a, _ := s.AddAsset(ctx, "SBER", "stocks")
	_, err := s.Save(ctx, "1", "", trade("buy", a.ID, "1", "1", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	buy, err := s.Save(ctx, "1", "", trade("buy", a.ID, "2", "2", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(ctx, "1", "", split(a.ID, "3", "1", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, "1", buy.ID); err == nil {
		t.Fatal("periodic deletion accepted")
	}
	if scalarString(t, store.DB(), "SELECT count(*) FROM operations") != "3" || scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02'") != "1" {
		t.Fatal("partial deletion")
	}
}
func TestSplitIntegrity(t *testing.T) {
	store, s, _ := setup(t)
	a, _ := s.AddAsset(ctx, "SBER", "stocks")
	buy, err := s.Save(ctx, "1", "", trade("buy", a.ID, "3", "1", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	op, err := s.Save(ctx, "1", "", split(a.ID, "3", "1", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []operations.Input{split("9999", "1", "10", "2026-01-02"), split(a.ID, "1", "1", "2026-01-02"), split(a.ID, "1.5", "10", "2026-01-02"), split(a.ID, "1", "10", "2026-02-30")} {
		if _, err = s.Save(ctx, "1", op.ID, in); err == nil {
			t.Fatal(in)
		}
	}
	if _, err = s.Save(ctx, "2", op.ID, op.Input); !errors.Is(err, operations.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "999", "", op.Input); !errors.Is(err, operations.ErrPortfolio) {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", buy.ID, op.Input); err == nil {
		t.Fatal("trade converted")
	}
	if _, err = s.Save(ctx, "1", op.ID, buy.Input); err == nil {
		t.Fatal("split converted")
	}
	for _, query := range []string{
		"INSERT INTO split_details VALUES (" + buy.ID + "," + a.ID + ",'1','10')",
		"INSERT INTO cash_flow_details VALUES (" + op.ID + ",'RUB','1','1')",
		"INSERT INTO trade_details VALUES (" + op.ID + "," + a.ID + ",'1','1','0','1')",
		"UPDATE operations SET kind='deposit' WHERE id=" + op.ID,
		"UPDATE operations SET kind='split' WHERE id=" + buy.ID,
		"UPDATE operations SET portfolio_id=2 WHERE id=" + op.ID,
		"UPDATE split_details SET operation_id=" + buy.ID + " WHERE operation_id=" + op.ID,
		"UPDATE split_details SET asset_id=9999 WHERE operation_id=" + op.ID,
	} {
		if _, err = store.DB().Exec(query); err == nil {
			t.Fatal(query)
		}
	}
	for _, field := range []string{"old_units", "new_units"} {
		for _, v := range []string{"0", "-1", "1.5", "1e3", "1/2", "", "01"} {
			if _, err = store.DB().Exec("UPDATE split_details SET "+field+"=? WHERE operation_id=?", v, op.ID); err == nil {
				t.Fatal(field, v)
			}
		}
	}
	if _, err = store.DB().Exec("UPDATE split_details SET new_units=old_units WHERE operation_id=?", op.ID); err == nil {
		t.Fatal("equal ratio")
	}
	if _, err = store.DB().Exec("DELETE FROM split_details WHERE operation_id=?", op.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.List(ctx, "1"); err == nil {
		t.Fatal("missing split details presented as zero")
	}
}
func TestConcurrentSplitWrites(t *testing.T) {
	store, s, _ := setup(t)
	a, _ := s.AddAsset(ctx, "SBER", "stocks")
	if _, err := s.Save(ctx, "1", "", trade("buy", a.ID, "1", "1", "0", "2026-01-01")); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.Save(ctx, "1", "", split(a.ID, "1", "2", "2026-01-02")); errs <- e }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02'") != "256" {
		t.Fatal("lost update")
	}
	snap, err := s.List(ctx, "1")
	if err != nil || len(snap.Operations) != 9 || snap.Operations[0].QuantityAfter != "256" || snap.Operations[7].QuantityBefore != "1" {
		t.Fatal(snap, err)
	}
}
func TestSplitMigrationFromTrades(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "rollback"}[fail], func(t *testing.T) {
			old := fstest.MapFS{}
			for _, name := range []string{"0001_schema_migrations.sql", "0002_currency_directory.sql", "0003_portfolios.sql", "0004_cash_flows.sql", "0005_unified_operations.sql"} {
				data, err := fs.ReadFile(migrations.Files, name)
				if err != nil {
					t.Fatal(err)
				}
				old[name] = &fstest.MapFile{Data: data}
			}
			path := filepath.Join(t.TempDir(), "legacy.sqlite3")
			store, err := storage.Open(ctx, path, old)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'Legacy','legacy','2026-01-01','RUB');
 INSERT INTO assets VALUES(1,'SBER');
 INSERT INTO operations VALUES(1,1,'deposit','2026-01-01','cash'),(2,1,'buy','2026-01-01','trade'),(100,1,'buy','2026-01-02','deleted');
 INSERT INTO cash_flow_details VALUES(1,'RUB','100','100');
 INSERT INTO trade_details VALUES(2,1,'12','24','1','2');
 DELETE FROM operations WHERE id=100;
 INSERT INTO daily_cash_balances VALUES(1,'2026-01-01','75');
 INSERT INTO daily_asset_balances VALUES(1,'2026-01-01',1,'12');`)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if _, err = store.DB().Exec("CREATE TRIGGER reject_6 BEFORE INSERT ON schema_migrations WHEN NEW.version=6 BEGIN SELECT RAISE(ABORT,'test'); END"); err != nil {
					t.Fatal(err)
				}
			}
			store.Close()
			store, err = storage.Open(ctx, path, migrations.Files)
			if fail {
				if err == nil {
					store.Close()
					t.Fatal("migration committed")
				}
				store, err = storage.Open(ctx, path, old)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if scalarString(t, store.DB(), "SELECT quantity FROM trade_details WHERE operation_id=2") != "12" || scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances") != "12" || scalarString(t, store.DB(), "SELECT seq FROM sqlite_sequence WHERE name='operations'") != "100" {
				t.Fatal("migration lost data")
			}
			copies, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
			if len(copies) != 1 {
				t.Fatal(copies)
			}
			if fail {
				if scalarString(t, store.DB(), "SELECT count(*) FROM sqlite_master WHERE name='split_details'") != "0" {
					t.Fatal("partial migration")
				}
				return
			}
			s := operations.NewService(NewRepository(store.DB()))
			snap, err := s.List(ctx, "1")
			if err != nil || snap.Balance != "75" || snap.Operations[0].ID != "2" {
				t.Fatal(snap, err)
			}
			op, err := s.Save(ctx, "1", "", split("1", "10", "1", "2026-01-02"))
			if err != nil || op.ID != "101" {
				t.Fatal(op, err)
			}
			if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02'") != "1.2" {
				t.Fatal("reverse split")
			}
		})
	}
}

func TestSplitEditingAssetAndPreviewOrder(t *testing.T) {
	store, s, _ := setup(t)
	a, _ := s.AddAsset(ctx, "A", "stocks")
	b, _ := s.AddAsset(ctx, "B", "stocks")
	for _, x := range []struct{ a, q string }{{a.ID, "12"}, {b.ID, "5"}} {
		if _, err := s.Save(ctx, "1", "", trade("buy", x.a, x.q, "1", "0", "2026-01-01")); err != nil {
			t.Fatal(err)
		}
	}
	first, err := s.Save(ctx, "1", "", split(a.ID, "1", "10", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Save(ctx, "1", "", split(a.ID, "10", "1", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewSplit(ctx, "1", first.ID, split(a.ID, "1", "2", "2026-01-02"))
	if err != nil || preview.Before != "12" || preview.After != "24" {
		t.Fatal(preview, err)
	}
	// An edit keeps its original position within the split group.
	if _, err = s.Save(ctx, "1", first.ID, split(a.ID, "1", "2", "2026-01-02")); err != nil {
		t.Fatal(err)
	}
	snap, err := s.List(ctx, "1")
	if err != nil || snap.Operations[0].ID != second.ID || snap.Operations[0].QuantityBefore != "24" || snap.Operations[0].QuantityAfter != "2.4" {
		t.Fatal(snap, err)
	}
	// Retargeting the second split restores A and derives B from its own history.
	if _, err = s.Save(ctx, "1", second.ID, split(b.ID, "10", "1", "2026-01-02")); err != nil {
		t.Fatal(err)
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02' AND asset_id=?", a.ID) != "24" || scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02' AND asset_id=?", b.ID) != "0.5" {
		t.Fatal("asset edit did not rebuild both assets")
	}
	sale, err := s.Save(ctx, "1", "", trade("sell", b.ID, "1", "1", "0", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || len(snap.Warnings) != 1 || snap.Warnings[0].OperationID != sale.ID || snap.Warnings[0].Available != "0.5" {
		t.Fatal(snap, err)
	}
	if err = s.Delete(ctx, "1", second.ID); err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || len(snap.Warnings) != 0 {
		t.Fatal(snap, err)
	}
	// Preview accepts exact downstream results and rejects edits of another family.
	if _, err = s.Save(ctx, "1", "", split(a.ID, "3", "1", "2026-01-03")); err != nil {
		t.Fatal(err)
	}
	_, err = s.PreviewSplit(ctx, "1", first.ID, split(a.ID, "5", "1", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	} // 12 / 5 / 3 = 0.8.
	_, err = s.PreviewSplit(ctx, "1", first.ID, split(a.ID, "1", "7", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	} // 84 / 3 = 28.
	if _, err = s.PreviewSplit(ctx, "1", sale.ID, split(b.ID, "1", "10", "2026-01-02")); err == nil {
		t.Fatal("trade previewed as split")
	}
	if _, err = s.PreviewSplit(ctx, "1", "9999", split(b.ID, "1", "10", "2026-01-02")); !errors.Is(err, operations.ErrNotFound) {
		t.Fatal(err)
	}
}
