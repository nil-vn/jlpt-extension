package flashcards_test

import (
	"context"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
	"github.com/nil-vn/jlpt-extension/app/internal/flashcards"
)

func TestRepositoryUpsertGetListAndCount(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()
	repo := flashcards.NewRepository(db)

	example := "例文"
	card := flashcards.Flashcard{
		ID:       "n5:vocabulary:neko",
		Level:    flashcards.LevelN5,
		Category: flashcards.CategoryVocabulary,
		Name:     "猫",
		Mean:     "cat",
		Hiragana: "ねこ",
		Example:  &example,
	}
	if err := repo.Upsert(ctx, card); err != nil {
		t.Fatalf("Upsert() insert error = %v", err)
	}

	got, found, err := repo.Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() found = false, want true")
	}
	if got.Name != card.Name || got.Example == nil || *got.Example != example {
		t.Fatalf("Get() = %+v, want inserted card", got)
	}

	card.Mean = "a cat"
	if err := repo.Upsert(ctx, card); err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("Count() = %d, want 1", count)
	}

	level := flashcards.LevelN5
	category := flashcards.CategoryVocabulary
	cards, err := repo.List(ctx, flashcards.Filter{Level: &level, Category: &category, Search: "cat"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(cards) != 1 || cards[0].Mean != "a cat" {
		t.Fatalf("List() = %+v, want updated card", cards)
	}
}
