package database_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
)

func TestOpenInDirCreatesDatabaseAndSchema(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	db, path, err := database.OpenInDir(ctx, dir)
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	if path != filepath.Join(dir, database.FileName) {
		t.Fatalf("path = %q, want %q", path, filepath.Join(dir, database.FileName))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}

	assertTableExists(t, db, "flashcards")
	assertTableExists(t, db, "app_settings")
	assertTableExists(t, db, "study_state")
}

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("third Migrate() error = %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = 1").Scan(&count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("schema_migrations version 1 rows = %d, want 1", count)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master for %s: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("table %s count = %d, want 1", table, count)
	}
}
