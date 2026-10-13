package sqlite_test

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"myinvest/internal/cashflow"
	. "myinvest/internal/cashflow/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"testing/fstest"
)

func trade(kind, aid, q, a, e, date string) cashflow.Input {
	return cashflow.Input{Kind: kind, AssetID: aid, Quantity: q, Amount: a, Expenses: e, Date: date, Comment: "Сделка"}
}
func scalarString(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var v string
	if err := db.QueryRow(query, args...).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestUnifiedTradesAndDailyBalances(t *testing.T) {
	store, s, path := setup(t)
	asset, err := s.AddAsset(ctx, " sber ")
	if err != nil || asset.Ticker != "SBER" {
		t.Fatal(asset, err)
	}
	if _, err = s.AddAsset(ctx, "sBeR"); err == nil {
		t.Fatal("duplicate ticker")
	}
	deposit, err := s.Save(ctx, "1", "", cashflow.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1000", PortfolioAmount: "1000"})
	if err != nil {
		t.Fatal(err)
	}
	buy, err := s.Save(ctx, "1", "", trade("buy", asset.ID, "2,5", "250", "5", "2026-01-01"))
	if err != nil || buy.AveragePrice != "100" {
		t.Fatal(buy, err)
	}
	snap, err := s.List(ctx, "1")
	if err != nil || snap.Balance != "745" || len(snap.Operations) != 2 {
		t.Fatal(snap, err)
	}
	sell, err := s.Save(ctx, "1", "", trade("sell", asset.ID, "1", "120", "2", "2026-01-02"))
	if err != nil || sell.AveragePrice != "120" {
		t.Fatal(sell, err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "863" || len(snap.Warnings) != 0 || snap.Operations[0].ID != sell.ID {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-02'"); v != "1.5" {
		t.Fatal(v)
	}
	if v := scalarString(t, store.DB(), "SELECT balance FROM daily_cash_balances WHERE portfolio_id=1 AND date='2026-01-01'"); v != "745" {
		t.Fatal(v)
	}
	if v := scalarString(t, store.DB(), "SELECT count(*) FROM daily_cash_balances WHERE portfolio_id=1"); v != "2" {
		t.Fatal(v)
	}
	// Same asset, independent portfolio and immutable operation family/portfolio.
	if _, err = s.Save(ctx, "2", "", trade("buy", asset.ID, "0.1", "1", "0", "2026-01-01")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "2", buy.ID, buy.Input); !errors.Is(err, cashflow.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", deposit.ID, buy.Input); err == nil {
		t.Fatal("cash converted into trade")
	}
	if _, err = s.Save(ctx, "1", buy.ID, deposit.Input); err == nil {
		t.Fatal("trade converted into cash")
	}
	for _, in := range []cashflow.Input{trade("buy", "999", "1", "1", "0", "2026-01-01"), trade("buy", asset.ID, "0", "1", "0", "2026-01-01"), trade("buy", asset.ID, "1", "1", "-1", "2026-01-01"), trade("buy", asset.ID, "1", "1", "0", "2026-02-30")} {
		if _, err = s.Save(ctx, "1", buy.ID, in); err == nil {
			t.Fatal(in)
		}
	}
	// Historical edit changes later warnings and both days, keeping the ID.
	buy, err = s.Save(ctx, "1", buy.ID, trade("buy", asset.ID, "0.2", "250", "5", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || len(snap.Warnings) != 1 || snap.Warnings[0].Available != "0.2" || snap.Warnings[0].Ticker != "SBER" {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE portfolio_id=1 AND date='2026-01-02'"); v != "-0.8" {
		t.Fatal(v)
	}
	// Moving the sale earlier warns against zero, and removes the abandoned date.
	sell, err = s.Save(ctx, "1", sell.ID, trade("sell", asset.ID, "1", "1", "2", "2025-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "744" || snap.Warnings[0].Available != "0" {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, store.DB(), "SELECT count(*) FROM daily_cash_balances WHERE portfolio_id=1 AND date='2026-01-02'"); v != "0" {
		t.Fatal(v)
	}
	// Deleting the deposit also changes derived cash, without affecting portfolio 2.
	if err = s.Delete(ctx, "1", deposit.ID); err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "-256" {
		t.Fatal(snap, err)
	}
	other, err := s.List(ctx, "2")
	if err != nil || other.Balance != "-1" {
		t.Fatal(other, err)
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
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "-256" || len(snap.Warnings) != 1 {
		t.Fatal(snap, err)
	}
	assets, err := s.ListAssets(ctx)
	if err != nil || len(assets) != 1 || assets[0] != asset {
		t.Fatal(assets, err)
	}
	for _, id := range []string{buy.ID, sell.ID} {
		if err = s.Delete(ctx, "1", id); err != nil {
			t.Fatal(err)
		}
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "0" || len(snap.Operations) != 0 {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, reopened.DB(), "SELECT count(*) FROM daily_cash_balances WHERE portfolio_id=1"); v != "0" {
		t.Fatal(v)
	}
	if v := scalarString(t, reopened.DB(), "SELECT count(*) FROM daily_asset_balances WHERE portfolio_id=1"); v != "0" {
		t.Fatal(v)
	}
}
func TestTradeUpdatesAssetDirectionPriceAndSameDayOrder(t *testing.T) {
	store, s, _ := setup(t)
	a, err := s.AddAsset(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.AddAsset(ctx, "B")
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Save(ctx, "1", "", trade("buy", a.ID, "3", "1", "0", "2026-01-01"))
	if err != nil || first.AveragePrice != "0.333333333333333333" || !first.Rounded {
		t.Fatal(first, err)
	}
	sell, err := s.Save(ctx, "1", "", trade("sell", a.ID, "1", "1", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.List(ctx, "1")
	if err != nil || len(snap.Warnings) != 0 {
		t.Fatal(snap, err)
	}
	_, err = s.Save(ctx, "1", first.ID, trade("sell", b.ID, "6", "1", "2", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "0" || len(snap.Warnings) != 2 || snap.Warnings[0].OperationID != first.ID || snap.Warnings[1].OperationID != sell.ID {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, store.DB(), "SELECT average_price FROM trade_details WHERE operation_id=?", first.ID); v != "0.166666666666666667" {
		t.Fatal(v)
	}
	if v := scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE asset_id=?", b.ID); v != "-6" {
		t.Fatal(v)
	}
}
func TestAtomicRebuildFailureAndIncompleteRead(t *testing.T) {
	store, s, _ := setup(t)
	asset, err := s.AddAsset(ctx, "SBER")
	if err != nil {
		t.Fatal(err)
	}
	op, err := s.Save(ctx, "1", "", trade("buy", asset.ID, "1", "10", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB().Exec("CREATE TRIGGER reject_totals BEFORE INSERT ON daily_asset_balances BEGIN SELECT RAISE(ABORT,'test failure'); END"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", op.ID} {
		if _, err = s.Save(ctx, "1", id, trade("buy", asset.ID, "2", "20", "3", "2026-01-02")); err == nil {
			t.Fatal("rebuild failure committed")
		}
	}
	if _, err = s.Save(ctx, "1", "", cashflow.Input{Kind: "deposit", Date: "2026-01-02", CurrencyCode: "RUB", Amount: "10", PortfolioAmount: "10"}); err == nil {
		t.Fatal("cash write ignored rebuild failure")
	}
	// Add a second operation before injecting a delete rebuild failure.
	if _, err = store.DB().Exec("DROP TRIGGER reject_totals"); err != nil {
		t.Fatal(err)
	}
	second, err := s.Save(ctx, "1", "", trade("buy", asset.ID, "1", "1", "0", "2026-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB().Exec("CREATE TRIGGER reject_totals BEFORE INSERT ON daily_asset_balances BEGIN SELECT RAISE(ABORT,'test failure'); END"); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, "1", second.ID); err == nil {
		t.Fatal("delete ignored rebuild failure")
	}
	snap, err := s.List(ctx, "1")
	if err != nil || len(snap.Operations) != 2 || snap.Balance != "-11" {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, store.DB(), "SELECT balance FROM daily_cash_balances WHERE date='2026-01-01'"); v != "-10" {
		t.Fatal(v)
	}
	if _, err = store.DB().Exec("DELETE FROM trade_details WHERE operation_id=?", op.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.List(ctx, "1"); err == nil {
		t.Fatal("incomplete operation accepted")
	}
}
func legacyFiles(t *testing.T) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	for _, name := range []string{"0001_schema_migrations.sql", "0002_currency_directory.sql", "0003_portfolios.sql", "0004_cash_flows.sql"} {
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = &fstest.MapFile{Data: data}
	}
	return out
}
func TestMigrationCashHistoryAndDeletedIDs(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "legacy.sqlite3")
			old := legacyFiles(t)
			store, err := storage.Open(ctx, path, old)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'Legacy','legacy','2026-01-01','RUB');
   INSERT INTO cash_flows VALUES(2,1,'deposit','2026-01-01','USD','1','0.1','old'),(10,1,'withdrawal','2026-01-01','USD','2','0.02','two'),(12,1,'deposit','2026-01-02','RUB','0.2','0.2','new'),(100,1,'deposit','2026-01-03','RUB','1','1','deleted');
   DELETE FROM cash_flows WHERE id=100;`); err != nil {
				t.Fatal(err)
			}
			if empty {
				if _, err = store.DB().Exec("DELETE FROM cash_flows"); err != nil {
					t.Fatal(err)
				}
			}
			store.Close()
			store, err = storage.Open(ctx, path, migrations.Files)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			s := cashflow.NewService(NewRepository(store.DB()))
			snap, err := s.List(ctx, "1")
			if err != nil {
				t.Fatal(err)
			}
			if empty {
				if snap.Balance != "0" || len(snap.Operations) != 0 {
					t.Fatal(snap)
				}
			} else {
				if snap.Balance != "0.28" || len(snap.Operations) != 3 || snap.Operations[1].ID != "10" || snap.Operations[2].Comment != "old" || snap.Operations[2].Amount != "1" || snap.Operations[2].CurrencyCode != "USD" {
					t.Fatal(snap)
				}
				if v := scalarString(t, store.DB(), "SELECT balance FROM daily_cash_balances WHERE date='2026-01-01'"); v != "0.08" {
					t.Fatal(v)
				}
			}
			if v := scalarString(t, store.DB(), "SELECT count(*) FROM assets"); v != "0" {
				t.Fatal(v)
			}
			op, err := s.Save(ctx, "1", "", cashflow.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1", PortfolioAmount: "1"})
			if err != nil {
				t.Fatal(err)
			}
			n, _ := strconv.Atoi(op.ID)
			if n <= 100 {
				t.Fatal("reused legacy high water ID", op)
			}
			if v := scalarString(t, store.DB(), "SELECT count(*) FROM sqlite_master WHERE name='cash_flows'"); v != "0" {
				t.Fatal(v)
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
			if v := scalarString(t, backup.DB(), "SELECT seq FROM sqlite_sequence WHERE name='cash_flows'"); v != "100" {
				t.Fatal(v)
			}
		})
	}
}
func TestMigrationExactDataStepRollback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite3")
	old := legacyFiles(t)
	store, err := storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'Legacy','legacy','2026-01-01','RUB');
 INSERT INTO cash_flows VALUES(1,1,'deposit','2026-01-01','RUB','1','1','');
 CREATE TRIGGER reject_5 BEFORE INSERT ON schema_migrations WHEN NEW.version=5 BEGIN SELECT RAISE(ABORT,'test'); END;`); err != nil {
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
	if v := scalarString(t, store.DB(), "SELECT amount FROM cash_flows WHERE id=1"); v != "1" {
		t.Fatal(v)
	}
	if v := scalarString(t, store.DB(), "SELECT count(*) FROM sqlite_master WHERE name IN ('operations','daily_cash_balances')"); v != "0" {
		t.Fatal(v)
	}
}
func TestConcurrentHistoryWrites(t *testing.T) {
	store, s, _ := setup(t)
	a, err := s.AddAsset(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Save(ctx, "1", "", trade("buy", a.ID, "0.1", "0.2", "0.01", "2026-01-01"))
			failures <- err
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	snap, err := s.List(ctx, "1")
	if err != nil || snap.Balance != "-4.2" || len(snap.Operations) != 20 {
		t.Fatal(snap, err)
	}
	if v := scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances"); v != "2" {
		t.Fatal(v)
	}
}

func TestDetailIntegrityAndCascade(t *testing.T) {
	store, s, _ := setup(t)
	a, err := s.AddAsset(ctx, "A")
	if err != nil {
		t.Fatal(err)
	}
	buy, err := s.Save(ctx, "1", "", trade("buy", a.ID, "1", "1", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	deposit, err := s.Save(ctx, "1", "", cashflow.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1", PortfolioAmount: "1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{buy.ID, "9999"} {
		if _, err = store.DB().Exec("INSERT INTO cash_flow_details VALUES (?,'RUB','1','1')", id); err == nil {
			t.Fatal("wrong cash detail", id)
		}
	}
	for _, id := range []string{deposit.ID, "9999"} {
		if _, err = store.DB().Exec("INSERT INTO trade_details VALUES (?,?,'1','1','0','1')", id, a.ID); err == nil {
			t.Fatal("wrong trade detail", id)
		}
	}
	for _, query := range []string{
		"UPDATE operations SET kind='deposit' WHERE id=" + buy.ID,
		"UPDATE operations SET portfolio_id=2 WHERE id=" + buy.ID,
		"UPDATE operations SET id=9999 WHERE id=" + buy.ID,
		"UPDATE trade_details SET operation_id=" + deposit.ID + " WHERE operation_id=" + buy.ID,
		"UPDATE cash_flow_details SET operation_id=" + buy.ID + " WHERE operation_id=" + deposit.ID,
		"INSERT INTO trade_details SELECT * FROM trade_details WHERE operation_id=" + buy.ID,
		"INSERT INTO cash_flow_details SELECT * FROM cash_flow_details WHERE operation_id=" + deposit.ID,
	} {
		if _, err = store.DB().Exec(query); err == nil {
			t.Fatal("integrity constraint missing", query)
		}
	}
	for _, field := range []string{"quantity", "amount", "expenses", "average_price"} {
		for _, value := range []string{"-1", "1e2", ".", "1..2", "", "1/2"} {
			if _, err = store.DB().Exec("UPDATE trade_details SET "+field+"=? WHERE operation_id=?", value, buy.ID); err == nil {
				t.Fatal("invalid decimal", field, value)
			}
		}
	}
	if _, err = store.DB().Exec("UPDATE trade_details SET asset_id=9999 WHERE operation_id=?", buy.ID); err == nil {
		t.Fatal("unknown asset")
	}
	if err = s.Delete(ctx, "1", buy.ID); err != nil {
		t.Fatal(err)
	}
	if v := scalarString(t, store.DB(), "SELECT count(*) FROM trade_details WHERE operation_id=?", buy.ID); v != "0" {
		t.Fatal(v)
	}
	if err = s.Delete(ctx, "1", deposit.ID); err != nil {
		t.Fatal(err)
	}
	if v := scalarString(t, store.DB(), "SELECT count(*) FROM cash_flow_details WHERE operation_id=?", deposit.ID); v != "0" {
		t.Fatal(v)
	}
}
