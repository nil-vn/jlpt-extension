package flashcards

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const MaxImportFileSizeBytes int64 = 25 * 1024 * 1024

var requiredImportFields = []string{"level", "category", "name", "mean", "hiragana", "image", "audio", "example"}

type ImportOptions struct {
	ReplaceLibrary bool `json:"replaceLibrary"`
	DryRun         bool `json:"dryRun"`
}

type ValidationError struct {
	Index   *int   `json:"index,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type ImportResult struct {
	BatchID      int64             `json:"batchId"`
	SourceSHA256 string            `json:"sourceSha256"`
	TotalRows    int               `json:"totalRows"`
	ValidRows    int               `json:"validRows"`
	InvalidRows  int               `json:"invalidRows"`
	Inserted     int               `json:"inserted"`
	Updated      int               `json:"updated"`
	Errors       []ValidationError `json:"errors"`
}

type ImportService struct {
	db *sql.DB
}

func NewImportService(db *sql.DB) *ImportService {
	return &ImportService{db: db}
}

func (s *ImportService) ImportFlashcardsFromFile(ctx context.Context, path string, options ImportOptions) (ImportResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("stat import file: %w", err)
	}
	if info.Size() > MaxImportFileSizeBytes {
		return ImportResult{}, fmt.Errorf("import file size %d exceeds limit %d", info.Size(), MaxImportFileSizeBytes)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("read import file: %w", err)
	}
	return s.ImportFlashcardsFromBytes(ctx, filepath.Base(path), content, options)
}

func (s *ImportService) ImportFlashcardsFromBytes(ctx context.Context, filename string, content []byte, options ImportOptions) (ImportResult, error) {
	if int64(len(content)) > MaxImportFileSizeBytes {
		return ImportResult{}, fmt.Errorf("import file size %d exceeds limit %d", len(content), MaxImportFileSizeBytes)
	}

	cards, result, err := ValidateImportDataset(content)
	if err != nil {
		return ImportResult{}, err
	}
	result.SourceSHA256 = hashBytes(content)
	if len(result.Errors) > 0 {
		return result, nil
	}

	if options.DryRun {
		inserted, updated, err := s.previewUpsertCounts(ctx, cards)
		if err != nil {
			return ImportResult{}, err
		}
		result.Inserted = inserted
		result.Updated = updated
		return result, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin import transaction: %w", err)
	}
	defer tx.Rollback()

	batchID, err := insertImportBatch(ctx, tx, filename, result.SourceSHA256, result.TotalRows, result.ValidRows, result.InvalidRows)
	if err != nil {
		return ImportResult{}, err
	}
	result.BatchID = batchID

	if options.ReplaceLibrary {
		if err := deleteFlashcardsNotInDataset(ctx, tx, cards); err != nil {
			return ImportResult{}, err
		}
	}

	for _, card := range cards {
		card.SourceBatchID = &batchID
		exists, err := flashcardExists(ctx, tx, card.ID)
		if err != nil {
			return ImportResult{}, err
		}
		if err := upsertFlashcard(ctx, tx, card); err != nil {
			return ImportResult{}, err
		}
		if exists {
			result.Updated++
		} else {
			result.Inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit import transaction: %w", err)
	}
	return result, nil
}

func ValidateImportDataset(content []byte) ([]Flashcard, ImportResult, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(content)), "[") {
		return nil, ImportResult{Errors: []ValidationError{{Message: "root JSON must be an array"}}, InvalidRows: 1}, nil
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(content, &rawItems); err != nil {
		return nil, ImportResult{}, fmt.Errorf("parse JSON array: %w", err)
	}
	result := ImportResult{TotalRows: len(rawItems)}
	cards := make([]Flashcard, 0, len(rawItems))
	seen := make(map[string]int, len(rawItems))

	for index, raw := range rawItems {
		card, itemErrors := validateImportItem(index, raw)
		if len(itemErrors) > 0 {
			result.Errors = append(result.Errors, itemErrors...)
			continue
		}
		card.ID = StableCardID(card)
		if previous, ok := seen[card.ID]; ok {
			result.Errors = append(result.Errors, fieldError(index, "name", fmt.Sprintf("duplicate card matches row %d", previous)))
			continue
		}
		seen[card.ID] = index
		cards = append(cards, card)
	}

	result.ValidRows = len(cards)
	result.InvalidRows = result.TotalRows - result.ValidRows
	return cards, result, nil
}

func validateImportItem(index int, raw json.RawMessage) (Flashcard, []ValidationError) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return Flashcard{}, []ValidationError{indexError(index, "item must be an object")}
	}

	errs := make([]ValidationError, 0)
	for _, field := range requiredImportFields {
		if _, ok := obj[field]; !ok {
			errs = append(errs, fieldError(index, field, "missing required field"))
		}
	}
	level, ok := decodeString(obj["level"])
	if !ok || !isValidLevel(Level(level)) {
		errs = append(errs, fieldError(index, "level", "must be one of n5, n4, n3, n2, n1"))
	}
	category, ok := decodeString(obj["category"])
	if !ok || !isValidCategory(Category(category)) {
		errs = append(errs, fieldError(index, "category", "must be one of gramma, vocabulary, kanji, reading, listening"))
	}

	name, ok := decodeString(obj["name"])
	if !ok {
		errs = append(errs, fieldError(index, "name", "must be a string"))
	} else if strings.TrimSpace(name) == "" {
		errs = append(errs, fieldError(index, "name", "must not be empty"))
	}
	mean, ok := decodeString(obj["mean"])
	if !ok {
		errs = append(errs, fieldError(index, "mean", "must be a string"))
	} else if strings.TrimSpace(mean) == "" {
		errs = append(errs, fieldError(index, "mean", "must not be empty"))
	}
	hiragana, ok := decodeString(obj["hiragana"])
	if !ok {
		errs = append(errs, fieldError(index, "hiragana", "must be a string"))
	}

	image, ok := decodeNullableString(obj["image"])
	if !ok {
		errs = append(errs, fieldError(index, "image", "must be a string or null"))
	}
	audio, ok := decodeNullableString(obj["audio"])
	if !ok {
		errs = append(errs, fieldError(index, "audio", "must be a string or null"))
	}
	example, ok := decodeNullableString(obj["example"])
	if !ok {
		errs = append(errs, fieldError(index, "example", "must be a string or null"))
	}

	if len(errs) > 0 {
		return Flashcard{}, errs
	}
	return Flashcard{Level: Level(level), Category: Category(category), Name: name, Mean: mean, Hiragana: hiragana, Image: image, Audio: audio, Example: example}, nil
}

func StableCardID(card Flashcard) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{string(card.Level), string(card.Category), card.Name, card.Hiragana, card.Mean}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func isValidLevel(level Level) bool {
	switch level {
	case LevelN5, LevelN4, LevelN3, LevelN2, LevelN1:
		return true
	default:
		return false
	}
}

func isValidCategory(category Category) bool {
	switch category {
	case CategoryGramma, CategoryVocabulary, CategoryKanji, CategoryReading, CategoryListening:
		return true
	default:
		return false
	}
}

func decodeString(raw json.RawMessage) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func decodeNullableString(raw json.RawMessage) (*string, bool) {
	if string(raw) == "null" {
		return nil, true
	}
	value, ok := decodeString(raw)
	if !ok {
		return nil, false
	}
	return &value, true
}

func indexError(index int, message string) ValidationError {
	return ValidationError{Index: &index, Message: message}
}

func fieldError(index int, field string, message string) ValidationError {
	return ValidationError{Index: &index, Field: field, Message: message}
}

func (s *ImportService) previewUpsertCounts(ctx context.Context, cards []Flashcard) (inserted int, updated int, err error) {
	for _, card := range cards {
		exists, err := flashcardExists(ctx, s.db, card.ID)
		if err != nil {
			return 0, 0, err
		}
		if exists {
			updated++
		} else {
			inserted++
		}
	}
	return inserted, updated, nil
}

type importDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func insertImportBatch(ctx context.Context, db importDB, filename, sha string, totalRows, validRows, invalidRows int) (int64, error) {
	res, err := db.ExecContext(ctx, `INSERT INTO import_batches (source_filename, source_sha256, total_rows, valid_rows, invalid_rows)
VALUES (?, ?, ?, ?, ?)`, filename, sha, totalRows, validRows, invalidRows)
	if err != nil {
		return 0, fmt.Errorf("insert import batch: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read import batch id: %w", err)
	}
	return id, nil
}

func flashcardExists(ctx context.Context, db importDB, id string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM flashcards WHERE id = ?", id).Scan(&count); err != nil {
		return false, fmt.Errorf("check flashcard %q exists: %w", id, err)
	}
	return count > 0, nil
}

func upsertFlashcard(ctx context.Context, db importDB, card Flashcard) error {
	_, err := db.ExecContext(ctx, `INSERT INTO flashcards (
  id, level, category, name, mean, hiragana, image, audio, example, source_batch_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  level = excluded.level,
  category = excluded.category,
  name = excluded.name,
  mean = excluded.mean,
  hiragana = excluded.hiragana,
  image = excluded.image,
  audio = excluded.audio,
  example = excluded.example,
  source_batch_id = excluded.source_batch_id,
  updated_at = CURRENT_TIMESTAMP`, card.ID, card.Level, card.Category, card.Name, card.Mean, card.Hiragana, card.Image, card.Audio, card.Example, card.SourceBatchID)
	if err != nil {
		return fmt.Errorf("upsert flashcard %q: %w", card.ID, err)
	}
	return nil
}


func deleteFlashcardsNotInDataset(ctx context.Context, db importDB, cards []Flashcard) error {
	if len(cards) == 0 {
		if _, err := db.ExecContext(ctx, "DELETE FROM flashcards"); err != nil {
			return fmt.Errorf("replace library delete flashcards: %w", err)
		}
		return nil
	}

	placeholders := make([]string, len(cards))
	args := make([]any, len(cards))
	for i, card := range cards {
		placeholders[i] = "?"
		args[i] = card.ID
	}
	query := fmt.Sprintf("DELETE FROM flashcards WHERE id NOT IN (%s)", strings.Join(placeholders, ","))
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("replace library delete removed cards: %w", err)
	}
	return nil
}
