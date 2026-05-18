package study_test

import (
	"context"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
	"github.com/nil-vn/jlpt-extension/app/internal/flashcards"
	"github.com/nil-vn/jlpt-extension/app/internal/study"
)

func TestRepositoryGetDefaultAndSave(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	cards := flashcards.NewRepository(db)
	cardID := "n5:gramma:test"
	if err := cards.Upsert(ctx, flashcards.Flashcard{ID: cardID, Level: flashcards.LevelN5, Category: flashcards.CategoryGramma, Name: "です", Mean: "to be", Hiragana: "です"}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	repo := study.NewRepository(db)
	state, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get() default error = %v", err)
	}
	if state.OrderMode != study.OrderModeSequential || state.RevealAnswers || state.CurrentCardID != nil {
		t.Fatalf("default state = %+v, want sequential unrevealed with no card", state)
	}

	state = study.State{CurrentCardID: &cardID, OrderMode: study.OrderModeRandom, RevealAnswers: true}
	if err := repo.Save(ctx, state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get() saved error = %v", err)
	}
	if got.CurrentCardID == nil || *got.CurrentCardID != cardID || got.OrderMode != study.OrderModeRandom || !got.RevealAnswers {
		t.Fatalf("Get() = %+v, want saved state", got)
	}
}
