package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetRaw(ctx context.Context, key string, value json.RawMessage) error {
	if !json.Valid(value) {
		return fmt.Errorf("setting %q contains invalid JSON", key)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO app_settings (key, value_json)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = CURRENT_TIMESTAMP`, key, string(value))
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

func (r *Repository) GetRaw(ctx context.Context, key string) (json.RawMessage, bool, error) {
	var raw string
	if err := r.db.QueryRowContext(ctx, "SELECT value_json FROM app_settings WHERE key = ?", key).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return json.RawMessage(raw), true, nil
}

func Set[T any](ctx context.Context, repo *Repository, key string, value T) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal setting %q: %w", key, err)
	}
	return repo.SetRaw(ctx, key, raw)
}

func Get[T any](ctx context.Context, repo *Repository, key string) (T, bool, error) {
	var zero T
	raw, found, err := repo.GetRaw(ctx, key)
	if err != nil || !found {
		return zero, found, err
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return zero, true, fmt.Errorf("unmarshal setting %q: %w", key, err)
	}
	return value, true, nil
}
