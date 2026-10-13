// Package migrations contains the versioned SQL and exact data steps shipped with the application.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	cashsqlite "myinvest/internal/cashflow/sqlite"
)

//go:embed *.sql
var scripts embed.FS

type bundle struct{ embed.FS }

// Files contains SQL and its transactional Go data migration hook.
var Files = bundle{scripts}

func (bundle) AfterMigration(ctx context.Context, version int, tx *sql.Tx) error {
	if version == 5 {
		return cashsqlite.RebuildAll(ctx, tx)
	}
	return nil
}
