package flashcards

import (
	"context"
	"database/sql"
	"fmt"
)

type Level string

type Category string

const (
	LevelN5 Level = "n5"
	LevelN4 Level = "n4"
	LevelN3 Level = "n3"
	LevelN2 Level = "n2"
	LevelN1 Level = "n1"

	CategoryGramma     Category = "gramma"
	CategoryVocabulary Category = "vocabulary"
	CategoryKanji      Category = "kanji"
	CategoryReading    Category = "reading"
	CategoryListening  Category = "listening"
)

type Flashcard struct {
	ID            string
	Level         Level
	Category      Category
	Name          string
	Mean          string
	Hiragana      string
	Image         *string
	Audio         *string
	Example       *string
	SourceBatchID *int64
}

type Filter struct {
	Level    *Level
	Category *Category
	Search   string
	Limit    int
	Offset   int
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, card Flashcard) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO flashcards (
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

func (r *Repository) Get(ctx context.Context, id string) (Flashcard, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, level, category, name, mean, hiragana, image, audio, example, source_batch_id
FROM flashcards WHERE id = ?`, id)
	card, err := scanFlashcard(row)
	if err == sql.ErrNoRows {
		return Flashcard{}, false, nil
	}
	if err != nil {
		return Flashcard{}, false, fmt.Errorf("get flashcard %q: %w", id, err)
	}
	return card, true, nil
}

func (r *Repository) List(ctx context.Context, filter Filter) ([]Flashcard, error) {
	query := `SELECT id, level, category, name, mean, hiragana, image, audio, example, source_batch_id FROM flashcards WHERE 1=1`
	args := make([]any, 0, 5)
	if filter.Level != nil {
		query += " AND level = ?"
		args = append(args, *filter.Level)
	}
	if filter.Category != nil {
		query += " AND category = ?"
		args = append(args, *filter.Category)
	}
	if filter.Search != "" {
		query += " AND (name LIKE ? OR mean LIKE ? OR hiragana LIKE ?)"
		like := "%" + filter.Search + "%"
		args = append(args, like, like, like)
	}
	query += " ORDER BY level, category, name"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list flashcards: %w", err)
	}
	defer rows.Close()

	cards := make([]Flashcard, 0)
	for rows.Next() {
		card, err := scanFlashcard(rows)
		if err != nil {
			return nil, fmt.Errorf("scan flashcard: %w", err)
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate flashcards: %w", err)
	}
	return cards, nil
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM flashcards").Scan(&count); err != nil {
		return 0, fmt.Errorf("count flashcards: %w", err)
	}
	return count, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFlashcard(row scanner) (Flashcard, error) {
	var card Flashcard
	var image, audio, example sql.NullString
	var batch sql.NullInt64
	if err := row.Scan(&card.ID, &card.Level, &card.Category, &card.Name, &card.Mean, &card.Hiragana, &image, &audio, &example, &batch); err != nil {
		return Flashcard{}, err
	}
	card.Image = nullableStringPtr(image)
	card.Audio = nullableStringPtr(audio)
	card.Example = nullableStringPtr(example)
	if batch.Valid {
		card.SourceBatchID = &batch.Int64
	}
	return card, nil
}

func nullableStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
