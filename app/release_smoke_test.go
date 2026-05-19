package main

import (
	"context"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
)

func TestReleaseSmokeImportRunAndStudy(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	service := NewAppService(db)
	status, err := service.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Name != appDisplayName || status.LibraryCount != 0 {
		t.Fatalf("initial status = %+v, want empty %q app", status, appDisplayName)
	}

	fixture := `[
		{"level":"n5","category":"vocabulary","name":"猫","mean":"cat","hiragana":"ねこ","image":null,"audio":null,"example":"猫が好きです。"},
		{"level":"n5","category":"kanji","name":"山","mean":"mountain","hiragana":"やま","image":null,"audio":null,"example":null}
	]`
	preview, err := service.PreviewImportJSON("release-smoke.json", fixture)
	if err != nil {
		t.Fatalf("PreviewImportJSON() error = %v", err)
	}
	if preview.ValidRows != 2 || preview.Inserted != 2 || preview.InvalidRows != 0 {
		t.Fatalf("preview = %+v, want dry-run import plan for 2 valid rows", preview)
	}
	status, err = service.Status()
	if err != nil {
		t.Fatalf("Status() after preview error = %v", err)
	}
	if status.LibraryCount != 0 {
		t.Fatalf("LibraryCount after dry-run preview = %d, want 0", status.LibraryCount)
	}

	result, err := service.ImportFlashcardsFromJSON("release-smoke.json", fixture, ImportOptions{})
	if err != nil {
		t.Fatalf("ImportFlashcardsFromJSON() error = %v", err)
	}
	if result.Inserted != 2 || result.Updated != 0 || result.InvalidRows != 0 {
		t.Fatalf("import result = %+v, want 2 inserted rows", result)
	}

	cards, err := service.ListFlashcards(FlashcardFilter{Level: "n5", Limit: 10})
	if err != nil {
		t.Fatalf("ListFlashcards() error = %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("ListFlashcards() returned %d cards, want 2", len(cards))
	}

	study, err := service.UpdateStudySettings(StudySettingsDTO{
		SelectedLevels:    []string{"n5"},
		EnabledCategories: []string{"vocabulary", "kanji"},
		OrderMode:         "sequential",
		DailyGoal:         5,
	})
	if err != nil {
		t.Fatalf("UpdateStudySettings() error = %v", err)
	}
	if study.CurrentCard == nil || study.TotalCards != 2 {
		t.Fatalf("study state = %+v, want imported cards available for run smoke", study)
	}
}
