package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	shipped "myinvest/migrations"
)

var ctx = context.Background()

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

func openTest(t *testing.T, path string, files fs.FS) *Store {
	t.Helper()
	store, err := Open(ctx, path, files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
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

func TestCreateReopenAndMigrationOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "данные #?", "myinvest.sqlite3")
	files := scripts(t, map[string]string{
		"0002_values.sql": "CREATE TABLE test_values (id INTEGER PRIMARY KEY, value TEXT NOT NULL); INSERT INTO test_values VALUES (1, 'initial');",
		"0003_update.sql": "UPDATE test_values SET value = 'kept' WHERE id = 1;",
	})
	store := openTest(t, path, files)
	if got := scalar(t, store.db, "SELECT count(*) FROM schema_migrations"); got != 3 {
		t.Fatalf("versions = %d", got)
	}
	execSQL(t, store.db, "INSERT INTO test_values VALUES (2, 'user data')")
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Ping(); err == nil {
		t.Fatal("connection remains open")
	}
	reopened := openTest(t, path, files)
	if got := scalar(t, reopened.db, "SELECT count(*) FROM test_values"); got != 2 {
		t.Fatalf("rows after restart = %d", got)
	}
	var value string
	if err := reopened.db.QueryRow("SELECT value FROM test_values WHERE id = 1").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "kept" {
		t.Fatalf("migration order: %s", value)
	}
	backups, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if len(backups) != 0 {
		t.Fatal("unnecessary backup on unchanged restart")
	}
}

func TestForeignKeysOnEveryConnection(t *testing.T) {
	store := openTest(t, filepath.Join(t.TempDir(), "db.sqlite3"), scripts(t, map[string]string{
		"0002_relations.sql": "CREATE TABLE test_parent(id INTEGER PRIMARY KEY); CREATE TABLE test_child(parent_id INTEGER NOT NULL REFERENCES test_parent(id));",
	}))
	var conns []*sql.Conn
	for range 3 {
		conn, err := store.db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
		defer conn.Close()
		var enabled int
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil || enabled != 1 {
			t.Fatalf("foreign_keys = %d: %v", enabled, err)
		}
		if _, err := conn.ExecContext(ctx, "INSERT INTO test_child VALUES (999)"); err == nil {
			t.Fatal("orphan accepted")
		}
	}
	execSQL(t, store.db, "INSERT INTO test_parent VALUES (1)")
	execSQL(t, store.db, "INSERT INTO test_child VALUES (1)")
	if _, err := store.db.Exec("DELETE FROM test_parent WHERE id = 1"); err == nil {
		t.Fatal("referenced parent deleted")
	}
	store.db.SetMaxIdleConns(0)
	if got := scalar(t, store.db, "PRAGMA foreign_keys"); got != 1 {
		t.Fatal("replacement connection has foreign keys disabled")
	}
}

func TestMigrationRollbackAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	first := scripts(t, nil)
	original := openTest(t, path, first)
	_ = original.Close()
	broken := scripts(t, map[string]string{"0002_broken.sql": "CREATE TABLE test_rolled_back(id INTEGER); INSERT INTO nonexistent VALUES (1);"})
	if store, err := Open(ctx, path, broken); err == nil || store != nil {
		t.Fatal("failed migration opened a ready store")
	}
	db, err := connect(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := scalar(t, db, "SELECT count(*) FROM sqlite_master WHERE name = 'test_rolled_back'"); got != 0 {
		t.Fatal("DDL did not roll back")
	}
	if got := scalar(t, db, "SELECT count(*) FROM schema_migrations"); got != 1 {
		t.Fatal("failed version recorded")
	}
	fixed := scripts(t, map[string]string{"0002_broken.sql": "CREATE TABLE test_rolled_back(id INTEGER);"})
	recovered := openTest(t, path, fixed)
	if got := scalar(t, recovered.db, "SELECT count(*) FROM schema_migrations"); got != 2 {
		t.Fatal("could not recover failed migration")
	}
}

func TestVersionRecordingIsAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	store := openTest(t, path, scripts(t, nil))
	execSQL(t, store.db, "CREATE TRIGGER test_reject_version BEFORE INSERT ON schema_migrations BEGIN SELECT RAISE(ABORT, 'test failure'); END;")
	_ = store.Close()
	_, err := Open(ctx, path, scripts(t, map[string]string{"0002_test.sql": "CREATE TABLE test_atomic(id INTEGER);"}))
	if err == nil {
		t.Fatal("version failure ignored")
	}
	db, err := connect(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := scalar(t, db, "SELECT count(*) FROM sqlite_master WHERE name = 'test_atomic'"); got != 0 {
		t.Fatal("schema committed without version")
	}
	if got := scalar(t, db, "SELECT count(*) FROM schema_migrations"); got != 1 {
		t.Fatal("version incorrectly recorded")
	}
}

func TestBackupIncludesWALBeforeMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	first := scripts(t, map[string]string{"0002_values.sql": "CREATE TABLE test_values(id INTEGER PRIMARY KEY);"})
	// Keep this connection open with checkpointing disabled: committed data lives in WAL.
	writer := openTest(t, path, first)
	writer.db.SetMaxOpenConns(1)
	execSQL(t, writer.db, "PRAGMA wal_autocheckpoint = 0")
	execSQL(t, writer.db, "INSERT INTO test_values VALUES (42)")
	info, err := os.Stat(path + "-wal")
	if err != nil || info.Size() == 0 {
		t.Fatalf("test needs populated WAL: %v", err)
	}
	newer := scripts(t, map[string]string{
		"0002_values.sql": "CREATE TABLE test_values(id INTEGER PRIMARY KEY);",
		"0003_values.sql": "INSERT INTO test_values VALUES (43);",
	})
	updated := openTest(t, path, newer)
	if got := scalar(t, updated.db, "SELECT count(*) FROM test_values"); got != 2 {
		t.Fatal("upgrade failed")
	}
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "*.sqlite3"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups: %v, %v", backups, err)
	}
	copy, err := connect(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer copy.Close()
	if got := scalar(t, copy, "SELECT count(*) FROM test_values"); got != 1 {
		t.Fatal("backup not from before migration")
	}
	if got := scalar(t, copy, "SELECT id FROM test_values"); got != 42 {
		t.Fatal("WAL data missing from backup")
	}
	if got := scalar(t, copy, "SELECT count(*) FROM schema_migrations"); got != 2 {
		t.Fatal("backup versions changed")
	}
	var integrity string
	if err := copy.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("backup integrity: %s %v", integrity, err)
	}
}

func TestBackupFailurePreventsMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	store := openTest(t, path, scripts(t, nil))
	_ = store.Close()
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "backups"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Open(ctx, path, scripts(t, map[string]string{"0002_test.sql": "CREATE TABLE test_not_run(id INTEGER);"}))
	if err == nil || !strings.Contains(err.Error(), "миграции не запускались") {
		t.Fatalf("expected backup failure: %v", err)
	}
	reopened := openTest(t, path, scripts(t, nil))
	if got := scalar(t, reopened.db, "SELECT count(*) FROM sqlite_master WHERE name = 'test_not_run'"); got != 0 {
		t.Fatal("migration ran despite backup failure")
	}
	if got := scalar(t, reopened.db, "SELECT count(*) FROM schema_migrations"); got != 1 {
		t.Fatal("history changed")
	}
}

func TestRejectChangedOrNewerHistory(t *testing.T) {
	for _, kind := range []string{"changed", "newer", "gap"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db.sqlite3")
			store := openTest(t, path, scripts(t, nil))
			files := scripts(t, nil)
			switch kind {
			case "changed":
				files["0001_schema_migrations.sql"].Data = append(files["0001_schema_migrations.sql"].Data, []byte("\n-- changed")...)
			case "newer":
				execSQL(t, store.db, "INSERT INTO schema_migrations VALUES (2, 'future', 'x', 'now')")
			case "gap":
				execSQL(t, store.db, "UPDATE schema_migrations SET version = 3 WHERE version = 1")
			}
			_ = store.Close()
			if _, err := Open(ctx, path, files); err == nil {
				t.Fatal("incompatible history accepted")
			}
		})
	}
}

func TestOpenErrorsPreserveFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db.sqlite3")
	content := []byte("not a database")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if store, err := Open(ctx, path, scripts(t, nil)); err == nil || store != nil {
		t.Fatal("invalid database accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(content) {
		t.Fatalf("original data changed: %v", err)
	}
	if _, err := Open(ctx, t.TempDir(), scripts(t, nil)); err == nil {
		t.Fatal("directory accepted as database")
	}
	if _, err := Open(ctx, "relative.sqlite3", scripts(t, nil)); err == nil {
		t.Fatal("relative database path accepted")
	}
}

func TestInvalidMigrationSequence(t *testing.T) {
	for _, name := range []string{"0003_gap.sql", "0001_duplicate.sql", "bad.sql"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db.sqlite3")
			if _, err := Open(ctx, path, scripts(t, map[string]string{name: "SELECT 1;"})); err == nil {
				t.Fatal("invalid sequence accepted")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("database created before migration validation")
			}
		})
	}
}

func TestDefaultPathOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MYINVEST_PORTABLE", "1")
	t.Setenv("MYINVEST_DATA_DIR", dir)
	path, err := DefaultPath()
	if err != nil || path != filepath.Join(dir, "myinvest.sqlite3") {
		t.Fatalf("path: %s %v", path, err)
	}
	t.Setenv("MYINVEST_DATA_DIR", "relative")
	if _, err := DefaultPath(); err == nil {
		t.Fatal("relative override accepted")
	}
}

func TestPortableDirectory(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, goos, executable, want string
	}{
		{"mac bundle", "darwin", filepath.Join(root, "myInvest.app", "Contents", "MacOS", "myInvest"), root},
		{"mac binary", "darwin", filepath.Join(root, "myInvest"), root},
		{"windows binary", "windows", filepath.Join(root, "myInvest.exe"), root},
		{"linux binary", "linux", filepath.Join(root, "myInvest"), root},
		{"non bundle", "darwin", filepath.Join(root, "other", "Contents", "MacOS", "myInvest"), filepath.Join(root, "other", "Contents", "MacOS")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := portableDirectory(tc.executable, tc.goos); got != tc.want {
				t.Fatalf("directory = %q, want %q", got, tc.want)
			}
		})
	}
	path := filepath.Join(portableDirectory(filepath.Join(root, "myInvest.app", "Contents", "MacOS", "myInvest"), "darwin"), "myinvest.sqlite3")
	store := openTest(t, path, scripts(t, nil))
	_ = store.Close()
	reopened := openTest(t, path, scripts(t, nil))
	if got := scalar(t, reopened.db, "SELECT count(*) FROM schema_migrations"); got != 1 {
		t.Fatal("portable database did not survive reopening")
	}
}

func TestDefaultPathPortableMode(t *testing.T) {
	t.Setenv("MYINVEST_DATA_DIR", "")
	t.Setenv("MYINVEST_PORTABLE", "")
	normal, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYINVEST_PORTABLE", "0")
	if got, err := DefaultPath(); err != nil || got != normal {
		t.Fatalf("disabled portable mode: %s %v", got, err)
	}
	t.Setenv("MYINVEST_PORTABLE", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(portableDirectory(executable, runtime.GOOS), "myinvest.sqlite3")
	if got, err := DefaultPath(); err != nil || got != want {
		t.Fatalf("portable path: %s %v, want %s", got, err, want)
	}
	t.Setenv("MYINVEST_PORTABLE", "yes")
	if _, err := DefaultPath(); err == nil {
		t.Fatal("invalid portable setting accepted")
	}
}

func TestWindowsDatabaseURI(t *testing.T) {
	uri := databaseURI("C:/Users/данные #?/myinvest.sqlite3", url.Values{"_pragma": {"foreign_keys(1)"}})
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Host != "" || parsed.Path != "/C:/Users/данные #?/myinvest.sqlite3" || parsed.Query().Get("_pragma") != "foreign_keys(1)" {
		t.Fatalf("invalid Windows URI: %s (%v)", uri, err)
	}
}
