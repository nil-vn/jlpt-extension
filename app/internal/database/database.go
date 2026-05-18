package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

const FileName = "jlpt.db"

type Options struct {
	Path          string
	SkipMigration bool
}

func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	if opts.Path == "" {
		path, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		opts.Path = path
	}

	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := configure(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	if !opts.SkipMigration {
		if err := Migrate(ctx, db); err != nil {
			db.Close()
			return nil, err
		}
	}

	return db, nil
}

func OpenInDir(ctx context.Context, dir string) (*sql.DB, string, error) {
	path := filepath.Join(dir, FileName)
	db, err := Open(ctx, Options{Path: path})
	if err != nil {
		return nil, "", err
	}
	return db, path, nil
}

func DefaultPath() (string, error) {
	base, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, FileName), nil
}

func dataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "JLPT Flashcard"), nil
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "JLPT Flashcard"), nil
	default:
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			return filepath.Join(xdgDataHome, "jlpt-flashcard"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	if home == "" {
		return "", errors.New("resolve user home: empty home directory")
	}
	return filepath.Join(home, ".local", "share", "jlpt-flashcard"), nil
}

func configure(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("configure sqlite %q: %w", pragma, err)
		}
	}
	return nil
}
