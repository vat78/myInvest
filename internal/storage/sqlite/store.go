// Package sqlite owns database connections, backups and schema migrations.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

// Store is ready for use only after all migrations have succeeded.
type Store struct{ db *sql.DB }

// DB provides the prepared pool to storage adapters. Store owns and closes it;
// callers must guard repository operations against application shutdown.
func (s *Store) DB() *sql.DB { return s.db }

// DefaultPath stores beside the application unless explicitly overridden.
// It is independent of the working directory and source checkout.
func DefaultPath() (string, error) {
	if override := os.Getenv("MYINVEST_DATA_DIR"); override != "" {
		if !filepath.IsAbs(override) {
			return "", fmt.Errorf("MYINVEST_DATA_DIR должен быть абсолютным")
		}
		return filepath.Join(override, "myinvest.sqlite3"), nil
	}
	switch os.Getenv("MYINVEST_PORTABLE") {
	case "0":
	case "", "1":
		executable, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("определить путь приложения: %w", err)
		}
		executable, err = filepath.EvalSymlinks(executable)
		if err != nil {
			return "", fmt.Errorf("определить фактический путь приложения: %w", err)
		}
		return filepath.Join(portableDirectory(executable, runtime.GOOS), "myinvest.sqlite3"), nil
	default:
		return "", fmt.Errorf("MYINVEST_PORTABLE должен быть 0 или 1")
	}
	var base string
	var err error
	switch runtime.GOOS {
	case "darwin":
		var home string
		home, err = os.UserHomeDir()
		base = filepath.Join(home, "Library", "Application Support")
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			err = fmt.Errorf("LOCALAPPDATA не задан")
		}
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			var home string
			home, err = os.UserHomeDir()
			base = filepath.Join(home, ".local", "share")
		}
	}
	if err != nil {
		return "", fmt.Errorf("определить каталог данных: %w", err)
	}
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("каталог данных должен быть абсолютным: %q", base)
	}
	return filepath.Join(base, "myInvest", "myinvest.sqlite3"), nil
}

func portableDirectory(executable, goos string) string {
	dir := filepath.Dir(executable)
	if goos == "darwin" && filepath.Base(dir) == "MacOS" {
		contents := filepath.Dir(dir)
		bundle := filepath.Dir(contents)
		if filepath.Base(contents) == "Contents" && strings.HasSuffix(filepath.Base(bundle), ".app") {
			return filepath.Dir(bundle)
		}
	}
	return dir
}

func Open(ctx context.Context, path string, migrations fs.FS) (*Store, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("путь базы должен быть абсолютным")
	}
	// Validate shipped migrations before changing any database files.
	scripts, err := loadMigrations(migrations)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	existing := err == nil
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("проверить базу %s: %w", path, err)
	}
	if existing && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("база %s не является обычным файлом", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("создать каталог базы: %w", err)
	}
	if !existing {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err != nil {
			return nil, fmt.Errorf("создать базу: %w", err)
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	}
	db, err := connect(path)
	if err != nil {
		return nil, fmt.Errorf("открыть базу %s: %w", path, err)
	}
	fail := func(err error) (*Store, error) { _ = db.Close(); return nil, err }
	if err := db.PingContext(ctx); err != nil {
		return fail(fmt.Errorf("открыть базу %s: %w", path, err))
	}
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		return fail(fmt.Errorf("настроить журнал базы: %w", err))
	}
	if err := migrate(ctx, db, path, existing, scripts); err != nil {
		return fail(err)
	}
	return &Store{db: db}, nil
}

func connect(path string) (*sql.DB, error) {
	params := url.Values{}
	params.Add("_pragma", "foreign_keys(1)")
	params.Add("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "synchronous(FULL)")
	// URL encoding also handles spaces, Cyrillic, '?' and '#' in user paths.
	return sql.Open("sqlite", databaseURI(filepath.ToSlash(path), params))
}

// Windows drive letters belong to the URI path, not its authority.
func databaseURI(path string, params url.Values) string {
	if len(path) >= 3 && path[1] == ':' && path[2] == '/' {
		path = "/" + path
	}
	uri := url.URL{Scheme: "file", Path: path, RawQuery: params.Encode()}
	return uri.String()
}

func (s *Store) Close() error { return s.db.Close() }
