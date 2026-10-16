package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	storage "myinvest/internal/storage/sqlite"
	"myinvest/migrations"
	"path/filepath"
	"testing"
	"testing/fstest"
)

type previousBundle struct{ fstest.MapFS }

func (previousBundle) AfterMigration(ctx context.Context, v int, tx *sql.Tx) error {
	return migrations.Files.AfterMigration(ctx, v, tx)
}

func TestUpgradeFromMainMigrations(t *testing.T) {
	ctx := context.Background()
	old := previousBundle{fstest.MapFS{}}
	files, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name() >= "0015" {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, file.Name())
		if err != nil {
			t.Fatal(err)
		}
		old.MapFS[file.Name()] = &fstest.MapFile{Data: data}
	}
	path := filepath.Join(t.TempDir(), "main.db")
	db, err := storage.Open(ctx, path, old)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.DB().Exec(`INSERT INTO benchmarks(id,name,name_key,sources) VALUES(1,'CPI','cpi','https://example.org/cpi')`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()
	db, err = storage.Open(ctx, path, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err = db.DB().QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); err != nil || version != 16 {
		t.Fatal(version, err)
	}
	var sources string
	if err = db.DB().QueryRow("SELECT sources FROM benchmarks WHERE id=1").Scan(&sources); err != nil || sources != "https://example.org/cpi" {
		t.Fatal("existing benchmark changed", sources, err)
	}
	var count int
	if err = db.DB().QueryRow("SELECT count(*) FROM currency_financial_plans").Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if err != nil || len(backups) != 1 {
		t.Fatal(backups, err)
	}
	backup, err := sql.Open("sqlite", backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err = backup.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); err != nil || version != 14 {
		t.Fatal("backup does not preserve main schema", version, err)
	}
}

func TestMigrationBackupAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "upgrade", true: "rollback"}[fail], func(t *testing.T) {
			ctx := context.Background()
			old := previousBundle{fstest.MapFS{}}
			files, e := fs.ReadDir(migrations.Files, ".")
			if e != nil {
				t.Fatal(e)
			}
			for _, f := range files {
				if f.Name() >= "0016" {
					continue
				}
				data, e := fs.ReadFile(migrations.Files, f.Name())
				if e != nil {
					t.Fatal(e)
				}
				old.MapFS[f.Name()] = &fstest.MapFile{Data: data}
			}
			path := filepath.Join(t.TempDir(), "migration.db")
			db, e := storage.Open(ctx, path, old)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = db.DB().Exec(`INSERT INTO portfolios VALUES(1,'P','p','2024-01-01','RUB');INSERT INTO benchmarks(id,name,name_key) VALUES(1,'CPI','cpi')`); e != nil {
				t.Fatal(e)
			}
			// The legacy source is kept; its exact initial capital and original snapshot survive.
			if _, e = db.DB().Exec(`INSERT INTO financial_plans(portfolio_id,revision,start_year,years,mode,goal,contribution,return_percent,inflation_percent,benchmark_id,allow_estimated,initial_source,initial_capital,initial_numerator,initial_denominator,original)
    VALUES(1,7,2026,1,'contribution','0','12.34','0','0',1,0,'portfolio','333.666666666666666667','1001','3','{"complete":true,"firstContribution":"12.34","rows":[],"points":[],"factors":[{"year":"2026","value":"1","source":"expected"}]}')`); e != nil {
				t.Fatal(e)
			}

			if fail {
				if _, e = db.DB().Exec(`CREATE TRIGGER fail_16 BEFORE INSERT ON schema_migrations WHEN NEW.version=16 BEGIN SELECT RAISE(ABORT,'test');END`); e != nil {
					t.Fatal(e)
				}
			}
			db.Close()
			db, e = storage.Open(ctx, path, migrations.Files)
			if fail {
				if e == nil {
					db.Close()
					t.Fatal("missing rollback")
				}
				db, e = storage.Open(ctx, path, old)
			}
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			var count int
			if e = db.DB().QueryRow("SELECT count(*) FROM sqlite_master WHERE name='currency_financial_plans'").Scan(&count); e != nil || (fail && count != 0) || (!fail && count != 1) {
				t.Fatal(count, e)
			}
			if e = db.DB().QueryRow("SELECT count(*) FROM portfolios WHERE id=1").Scan(&count); e != nil || count != 1 {
				t.Fatal("existing data changed", e)
			}
			var original, num, den string
			var revision int
			if e = db.DB().QueryRow("SELECT original,initial_numerator,initial_denominator,revision FROM financial_plans WHERE portfolio_id=1").Scan(&original, &num, &den, &revision); e != nil || revision != 7 || num != "1001" || den != "3" {
				t.Fatal("legacy plan changed", e)
			}
			if !fail {
				var copied string
				if e = db.DB().QueryRow("SELECT original,initial_numerator,initial_denominator,revision FROM currency_financial_plans WHERE currency_code='RUB'").Scan(&copied, &num, &den, &revision); e != nil || copied != original || num != "1001" || den != "3" || revision != 7 {
					t.Fatal("plan not preserved", e)
				}
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
			if e = backup.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&count); e != nil || count != 15 {
				t.Fatal(count, e)
			}
		})
	}
}
