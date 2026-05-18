package migrations

import "embed"

// Files embeds the SQL migrations used to create and upgrade the desktop SQLite schema.
//
//go:embed *.sql
var Files embed.FS
