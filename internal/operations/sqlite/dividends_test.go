package sqlite_test

import (
	"errors"
	"io/fs"
	"myinvest/internal/operations"
	. "myinvest/internal/operations/sqlite"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"
)

func dividend(amount, asset, date string) operations.Input {
	return operations.Input{Kind: "dividend", Amount: amount, AssetID: asset, Date: date, Comment: "Зачисление"}
}
func TestDividendLifecycle(t *testing.T) {
	store, s, path := setup(t)
	asset, err := s.AddAsset(ctx, "SBER", "stocks")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "1", "", operations.Input{Kind: "deposit", Date: "2026-01-01", CurrencyCode: "RUB", Amount: "1000", PortfolioAmount: "1000"}); err != nil {
		t.Fatal(err)
	}
	buy, err := s.Save(ctx, "1", "", trade("buy", asset.ID, "2.5", "250", "0", "2026-01-01"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Save(ctx, "1", "", dividend("0,1", asset.ID, "2026-01-02"))
	if err != nil || a.Ticker != "SBER" || a.Amount != "0.1" {
		t.Fatal(a, err)
	}
	b, err := s.Save(ctx, "1", "", dividend("0.2", "", "2026-01-02"))
	if err != nil || b.AssetID != "" || b.Ticker != "" {
		t.Fatal(b, err)
	}
	if scalarString(t, store.DB(), "SELECT asset_id IS NULL FROM dividend_details WHERE operation_id=?", b.ID) != "1" {
		t.Fatal("empty asset must be NULL")
	}
	snap, err := s.List(ctx, "1")
	if err != nil || snap.Balance != "750.3" {
		t.Fatal(snap, err)
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02' AND portfolio_id=1") != "2.5" || scalarString(t, store.DB(), "SELECT price_operation_id FROM daily_asset_balances WHERE date='2026-01-02' AND portfolio_id=1") != buy.ID || scalarString(t, store.DB(), "SELECT deposits_rub FROM daily_cash_balances WHERE date='2026-01-02' AND portfolio_id=1") != "1000" {
		t.Fatal("dividend changed positions or deposits")
	}
	for _, in := range []operations.Input{dividend("0", "", "2026-01-02"), dividend("-1", "", "2026-01-02"), dividend("1", "999", "2026-01-02"), dividend("1", "", "2026-02-30"), {Kind: "deposit", Date: "2026-01-02", CurrencyCode: "RUB", Amount: "1", PortfolioAmount: "1"}} {
		if _, err = s.Save(ctx, "1", a.ID, in); err == nil {
			t.Fatal("accepted invalid edit", in)
		}
	}
	if _, err = s.Save(ctx, "2", a.ID, a.Input); !errors.Is(err, operations.ErrNotFound) {
		t.Fatal(err)
	}
	a, err = s.Save(ctx, "1", a.ID, dividend("1.00000000000000000001", "", "2025-12-01"))
	if err != nil || a.AssetID != "" || a.Ticker != "" {
		t.Fatal(a, err)
	}
	b, err = s.Save(ctx, "1", b.ID, dividend("0.2", asset.ID, "2026-01-02"))
	if err != nil || b.Ticker != "SBER" {
		t.Fatal(b, err)
	}
	if _, err = s.Save(ctx, "2", "", dividend("5", "", "2027-01-01")); err != nil {
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
	s = operations.NewService(NewRepository(reopened.DB()))
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "751.20000000000000000001" {
		t.Fatal(snap, err)
	}
	if err = s.Delete(ctx, "1", a.ID); err != nil {
		t.Fatal(err)
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "750.2" {
		t.Fatal(snap, err)
	}
	if scalarString(t, reopened.DB(), "SELECT count(*) FROM dividend_details WHERE operation_id=?", a.ID) != "0" || scalarString(t, reopened.DB(), "SELECT count(*) FROM daily_cash_balances WHERE portfolio_id=1 AND date='2025-12-01'") != "0" {
		t.Fatal("delete did not cascade or rebuild")
	}
	other, err := s.List(ctx, "2")
	if err != nil || other.Balance != "5" {
		t.Fatal(other, err)
	}
	// Force a detail failure after inserting the parent; transaction rolls everything back.
	if _, err = reopened.DB().Exec(`CREATE TRIGGER reject_dividend BEFORE INSERT ON dividend_details BEGIN SELECT RAISE(ABORT,'test'); END`); err != nil {
		t.Fatal(err)
	}
	before := scalarString(t, reopened.DB(), "SELECT count(*) FROM operations")
	if _, err = s.Save(ctx, "1", "", dividend("10", "", "2026-01-03")); err == nil {
		t.Fatal("missing storage failure")
	}
	if scalarString(t, reopened.DB(), "SELECT count(*) FROM operations") != before {
		t.Fatal("partial parent persisted")
	}
	snap, err = s.List(ctx, "1")
	if err != nil || snap.Balance != "750.2" {
		t.Fatal(snap, err)
	}
}

func TestDividendMigrationPreservesHistoryAndConstraints(t *testing.T) {
	old := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		version, _ := strconv.Atoi(name[:4])
		if version >= 15 {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		old[name] = &fstest.MapFile{Data: data}
	}
	path := filepath.Join(t.TempDir(), "migration.sqlite3")
	store, err := storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB().Exec(`INSERT INTO portfolios VALUES(1,'Test','test','2026-01-01','RUB');
 INSERT INTO assets(id,ticker,type_code) VALUES(1,'SBER','stocks');
 INSERT INTO operations(id,portfolio_id,kind,date,comment) VALUES(1,1,'deposit','2026-01-01',''),(2,1,'buy','2026-01-01',''),(3,1,'split','2026-01-02','');
 INSERT INTO cash_flow_details VALUES(1,'RUB','1000','1000');
 INSERT INTO trade_details VALUES(2,1,'3','300','0','100');
 INSERT INTO split_details VALUES(3,1,'1','2');
 UPDATE sqlite_sequence SET seq=100 WHERE name='operations';`)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := store.DB().Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = RebuildAll(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := operations.NewService(NewRepository(store.DB()))
	snap, err := s.List(ctx, "1")
	if err != nil || snap.Balance != "700" || len(snap.Operations) != 3 {
		t.Fatal(snap, err)
	}
	if scalarString(t, store.DB(), "SELECT quantity FROM daily_asset_balances WHERE date='2026-01-02'") != "6" || scalarString(t, store.DB(), "SELECT price_operation_id FROM daily_asset_balances WHERE date='2026-01-02'") != "2" || scalarString(t, store.DB(), "SELECT value_numerator FROM daily_asset_balances WHERE date='2026-01-02'") != "300" || scalarString(t, store.DB(), "SELECT seq FROM sqlite_sequence WHERE name='operations'") != "100" {
		t.Fatal("migration lost derived data or sequence")
	}
	op, err := s.Save(ctx, "1", "", dividend("0.1", "", "2026-01-03"))
	if err != nil || op.ID != "101" {
		t.Fatal(op, err)
	}
	for _, query := range []string{
		`INSERT INTO dividend_details VALUES(1,NULL,'1')`,
		`INSERT INTO cash_flow_details VALUES(101,'RUB','1','1')`,
		`INSERT INTO trade_details VALUES(101,1,'1','1','0','1')`,
		`INSERT INTO split_details VALUES(101,1,'1','2')`,
		`UPDATE dividend_details SET asset_id=999 WHERE operation_id=101`,
		`UPDATE dividend_details SET amount='0' WHERE operation_id=101`,
		`UPDATE dividend_details SET amount='-1' WHERE operation_id=101`,
		`UPDATE dividend_details SET operation_id=2 WHERE operation_id=101`,
		`UPDATE operations SET kind='deposit' WHERE id=101`,
		`UPDATE operations SET kind='dividend' WHERE id=1`,
	} {
		if _, err = store.DB().Exec(query); err == nil {
			t.Fatal("constraint missing", query)
		}
	}
}
