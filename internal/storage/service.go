package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"jlpt-extension/internal/flashcards"

	_ "github.com/mattn/go-sqlite3"
)

const databaseFileName = "jlpt-study-companion.db"

type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type Stats struct {
	DatabasePath string `json:"databasePath"`
	Cards        int    `json:"cards"`
}

// StoreService owns the application SQLite database. Phase 1 exposes the
// minimum lifecycle and import surface needed by the Wails frontend; later
// phases can add settings, scheduler state, notes, and bookmarks here.
type StoreService struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewStoreService() *StoreService {
	return &StoreService{}
}

func (s *StoreService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}

	err := s.db.Close()
	s.db = nil
	return err
}

func (s *StoreService) Open(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return fmt.Errorf("open sqlite database: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return fmt.Errorf("configure sqlite database: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db = db
	return nil
}

func (s *StoreService) ImportDatasetJSON(raw string) (ImportResult, error) {
	var cards []flashcards.Card
	if err := json.Unmarshal([]byte(raw), &cards); err != nil {
		return ImportResult{}, fmt.Errorf("parse dataset json: %w", err)
	}

	s.mu.RLock()
	db := s.db
	s.mu.RUnlock()
	if db == nil {
		return ImportResult{}, errors.New("database is not open")
	}

	tx, err := db.Begin()
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin import transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO cards (id, level, category, name, mean, hiragana, image, audio, example, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			level = excluded.level,
			category = excluded.category,
			name = excluded.name,
			mean = excluded.mean,
			hiragana = excluded.hiragana,
			image = excluded.image,
			audio = excluded.audio,
			example = excluded.example,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return ImportResult{}, fmt.Errorf("prepare card import: %w", err)
	}
	defer stmt.Close()

	result := ImportResult{}
	now := time.Now().UTC().Format(time.RFC3339)
	for index, card := range cards {
		if err := card.Validate(); err != nil {
			result.Skipped++
			continue
		}

		if _, err := stmt.Exec(
			card.ID(),
			card.Level,
			card.Category,
			card.Name,
			card.Mean,
			card.Hiragana,
			nullString(card.Image),
			nullString(card.Audio),
			nullString(card.Example),
			now,
		); err != nil {
			return ImportResult{}, fmt.Errorf("import card %d: %w", index+1, err)
		}
		result.Imported++
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit import transaction: %w", err)
	}

	return result, nil
}

func (s *StoreService) Stats() (Stats, error) {
	s.mu.RLock()
	db := s.db
	s.mu.RUnlock()
	if db == nil {
		return Stats{}, errors.New("database is not open")
	}

	var stats Stats
	if err := db.QueryRow(`PRAGMA database_list`).Scan(new(int), new(string), &stats.DatabasePath); err != nil {
		return Stats{}, fmt.Errorf("read database path: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&stats.Cards); err != nil {
		return Stats{}, fmt.Errorf("count cards: %w", err)
	}
	return stats, nil
}

func DefaultDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(configDir, "JLPT Study Companion", databaseFileName), nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cards (
			id TEXT PRIMARY KEY,
			level TEXT NOT NULL,
			category TEXT NOT NULL,
			name TEXT NOT NULL,
			mean TEXT NOT NULL,
			hiragana TEXT NOT NULL,
			image TEXT,
			audio TEXT,
			example TEXT,
			updated_at TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_cards_level_category ON cards(level, category);
	`)
	if err != nil {
		return fmt.Errorf("migrate sqlite database: %w", err)
	}
	return nil
}

func nullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}
