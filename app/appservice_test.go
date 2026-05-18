package main

import (
	"context"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
	"github.com/nil-vn/jlpt-extension/app/internal/flashcards"
)

func TestAppServiceStudyNavigationSettingsNotesAndBookmarks(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	repo := flashcards.NewRepository(db)
	cards := []flashcards.Flashcard{
		{ID: "n5:vocabulary:neko", Level: flashcards.LevelN5, Category: flashcards.CategoryVocabulary, Name: "猫", Mean: "cat", Hiragana: "ねこ"},
		{ID: "n5:vocabulary:inu", Level: flashcards.LevelN5, Category: flashcards.CategoryVocabulary, Name: "犬", Mean: "dog", Hiragana: "いぬ"},
		{ID: "n4:kanji:yama", Level: flashcards.LevelN4, Category: flashcards.CategoryKanji, Name: "山", Mean: "mountain", Hiragana: "やま"},
	}
	for _, card := range cards {
		if err := repo.Upsert(ctx, card); err != nil {
			t.Fatalf("Upsert(%q) error = %v", card.ID, err)
		}
	}

	service := NewAppService(db)
	state, err := service.UpdateStudySettings(StudySettingsDTO{
		SelectedLevels:    []string{"n5"},
		EnabledCategories: []string{"vocabulary"},
		OrderMode:         "sequential",
		RevealAnswers:     false,
		DailyGoal:         5,
	})
	if err != nil {
		t.Fatalf("UpdateStudySettings() error = %v", err)
	}
	if state.TotalCards != 2 || state.CurrentCard == nil || state.CurrentCard.ID != "n5:vocabulary:inu" {
		t.Fatalf("UpdateStudySettings() = %+v, want first filtered card", state)
	}

	state, err = service.MoveNext()
	if err != nil {
		t.Fatalf("MoveNext() error = %v", err)
	}
	if state.CurrentCard == nil || state.CurrentCard.ID != "n5:vocabulary:neko" {
		t.Fatalf("MoveNext() current = %+v, want next sequential card", state.CurrentCard)
	}

	state, err = service.MovePrevious()
	if err != nil {
		t.Fatalf("MovePrevious() error = %v", err)
	}
	if state.CurrentCard == nil || state.CurrentCard.ID != "n5:vocabulary:inu" {
		t.Fatalf("MovePrevious() current = %+v, want previous sequential card", state.CurrentCard)
	}

	card, err := service.SaveNote(state.CurrentCard.ID, "common animal")
	if err != nil {
		t.Fatalf("SaveNote() error = %v", err)
	}
	if card.Note != "common animal" {
		t.Fatalf("SaveNote() note = %q, want saved note", card.Note)
	}

	card, err = service.ToggleBookmark(state.CurrentCard.ID)
	if err != nil {
		t.Fatalf("ToggleBookmark() error = %v", err)
	}
	if !card.Bookmarked {
		t.Fatal("ToggleBookmark() bookmarked = false, want true")
	}
}
