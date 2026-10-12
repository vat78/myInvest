// Package migrations contains the versioned SQL shipped with the application.
package migrations

import "embed"

// Files is available without access to the source checkout.
//
//go:embed *.sql
var Files embed.FS
