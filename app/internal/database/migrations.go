package database

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nil-vn/jlpt-extension/app/migrations"
)

type migration struct {
	Version int
	Name    string
	SQL     string
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	items, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range items {
		applied, err := migrationApplied(ctx, db, m.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return err
		}
	}

	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	items := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", entry.Name(), err)
		}
		content, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		items = append(items, migration{Version: version, Name: entry.Name(), SQL: string(content)})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Version < items[j].Version })
	return items, nil
}

func migrationApplied(ctx context.Context, db *sql.DB, version int) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %d: %w", version, err)
	}
	return count > 0, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", m.Version, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("apply migration %s: %w", m.Name, err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES (?)", m.Version); err != nil {
		return fmt.Errorf("record migration %d: %w", m.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", m.Version, err)
	}
	return nil
}
