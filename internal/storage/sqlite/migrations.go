package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type migration struct {
	version                int
	after                  func(context.Context, int, *sql.Tx) error
	name, script, checksum string
}

var migrationName = regexp.MustCompile(`^([0-9]{4})_[a-z0-9_]+\.sql$`)

func loadMigrations(files fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("прочитать миграции: %w", err)
	}
	var result []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		match := migrationName.FindStringSubmatch(entry.Name())
		if match == nil {
			return nil, fmt.Errorf("неверное имя миграции: %s", entry.Name())
		}
		version, _ := strconv.Atoi(match[1])
		if version != len(result)+1 {
			return nil, fmt.Errorf("нарушен порядок миграций: %s", entry.Name())
		}
		data, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("прочитать %s: %w", entry.Name(), err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return nil, fmt.Errorf("пустая миграция: %s", entry.Name())
		}
		var after func(context.Context, int, *sql.Tx) error
		if hook, ok := files.(interface {
			AfterMigration(context.Context, int, *sql.Tx) error
		}); ok {
			after = hook.AfterMigration
		}
		result = append(result, migration{version: version, name: entry.Name(), script: string(data), checksum: fmt.Sprintf("%x", sha256.Sum256(data)), after: after})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("SQL-миграции отсутствуют")
	}
	return result, nil
}

func appliedCount(ctx context.Context, db *sql.DB, scripts []migration) (int, error) {
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'").Scan(&exists); err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT version, name, checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var version int
		var name, checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			return 0, err
		}
		if count >= len(scripts) {
			return 0, fmt.Errorf("база создана более новой версией приложения")
		}
		want := scripts[count]
		if version != want.version || name != want.name || checksum != want.checksum {
			return 0, fmt.Errorf("история миграций не совпадает с приложением: версия %d", version)
		}
		count++
	}
	return count, rows.Err()
}

func migrate(ctx context.Context, db *sql.DB, path string, existing bool, scripts []migration) error {
	count, err := appliedCount(ctx, db, scripts)
	if err != nil {
		return fmt.Errorf("проверить историю миграций: %w", err)
	}
	if count == len(scripts) {
		return nil
	}
	if existing {
		if err := backup(ctx, db, path); err != nil {
			return fmt.Errorf("резервная копия не создана; миграции не запускались: %w", err)
		}
	}
	for _, script := range scripts[count:] {
		if err := apply(ctx, db, script); err != nil {
			return fmt.Errorf("миграция %s отменена: %w", script.name, err)
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, script migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, script.script); err != nil {
		return err
	}
	if script.after != nil {
		if err := script.after(ctx, script.version, tx); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES (?, ?, ?, ?)", script.version, script.name, script.checksum, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func backup(ctx context.Context, db *sql.DB, path string) error {
	dir := filepath.Join(filepath.Dir(path), "backups")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	// Reserve a unique name. SQLite permits an existing empty destination file.
	file, err := os.CreateTemp(dir, "myinvest-"+time.Now().UTC().Format("20060102T150405Z")+"-*.sqlite3")
	if err != nil {
		return err
	}
	destination := file.Name()
	if err := file.Close(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", destination); err != nil {
		// Only remove our incomplete backup, never the source database.
		_ = os.Remove(destination)
		return err
	}
	return nil
}
