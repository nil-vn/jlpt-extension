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

type recordingNotifier struct {
	payloads []NotificationPayload
}

func (n *recordingNotifier) ShowFlashcard(_ context.Context, payload NotificationPayload) error {
	n.payloads = append(n.payloads, payload)
	return nil
}

func TestAppServiceNotificationsSettingsPauseResumeAndContent(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	example := "猫が好きです。"
	repo := flashcards.NewRepository(db)
	cards := []flashcards.Flashcard{
		{ID: "n5:vocabulary:inu", Level: flashcards.LevelN5, Category: flashcards.CategoryVocabulary, Name: "犬", Mean: "dog", Hiragana: "いぬ"},
		{ID: "n5:vocabulary:neko", Level: flashcards.LevelN5, Category: flashcards.CategoryVocabulary, Name: "猫", Mean: "cat", Hiragana: "ねこ", Example: &example},
	}
	for _, card := range cards {
		if err := repo.Upsert(ctx, card); err != nil {
			t.Fatalf("Upsert(%q) error = %v", card.ID, err)
		}
	}

	notifier := &recordingNotifier{}
	service := NewAppServiceWithNotifier(db, notifier)
	state, err := service.UpdateStudySettings(StudySettingsDTO{
		SelectedLevels:              []string{"n5"},
		EnabledCategories:           []string{"vocabulary"},
		OrderMode:                   "sequential",
		DailyGoal:                   5,
		NotificationEnabled:         true,
		NotificationPaused:          false,
		NotificationIntervalMinutes: 0.5,
	})
	if err != nil {
		t.Fatalf("UpdateStudySettings() error = %v", err)
	}
	if !state.Settings.NotificationEnabled || state.Settings.NotificationPaused || state.Settings.NotificationIntervalMinutes != 0.5 {
		t.Fatalf("notification settings = %+v, want enabled/unpaused/0.5", state.Settings)
	}

	payload, err := service.ShowStudyNotificationNow()
	if err != nil {
		t.Fatalf("ShowStudyNotificationNow() error = %v", err)
	}
	if len(notifier.payloads) != 1 {
		t.Fatalf("notifier payload count = %d, want 1", len(notifier.payloads))
	}
	if payload.Title != "[N5 - Vocabulary]  犬 (いぬ)" || payload.Message != "dog" {
		t.Fatalf("payload = %+v, want extension-compatible title/message", payload)
	}

	state, err = service.SetNotificationPaused(true)
	if err != nil {
		t.Fatalf("SetNotificationPaused(true) error = %v", err)
	}
	if !state.Settings.NotificationPaused {
		t.Fatalf("NotificationPaused = false, want true")
	}
	if _, err := service.ShowStudyNotificationNow(); err == nil {
		t.Fatal("ShowStudyNotificationNow() error = nil, want paused error")
	}

	state, err = service.SetNotificationPaused(false)
	if err != nil {
		t.Fatalf("SetNotificationPaused(false) error = %v", err)
	}
	if state.Settings.NotificationPaused {
		t.Fatalf("NotificationPaused = true, want false")
	}
}
