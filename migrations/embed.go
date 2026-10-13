// Package migrations contains the versioned SQL and exact data steps shipped with the application.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	operationssqlite "myinvest/internal/operations/sqlite"
)

//go:embed *.sql
var scripts embed.FS

type bundle struct{ embed.FS }

// Files contains SQL and its transactional Go data migration hook.
var Files = bundle{scripts}

func (bundle) AfterMigration(ctx context.Context, version int, tx *sql.Tx) error {
	if version == 5 {
		return operationssqlite.RebuildAll(ctx, tx)
	}
	return nil
}
