package storage

import (
	"path/filepath"
	"testing"
)

func TestImportDatasetJSONStoresValidCards(t *testing.T) {
	service := NewStoreService()
	dbPath := filepath.Join(t.TempDir(), "cards.db")
	if err := service.Open(dbPath); err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = service.Close() }()

	result, err := service.ImportDatasetJSON(`[
		{"level":"n2","category":"vocabulary","name":"曖昧","hiragana":"あいまい","mean":"mơ hồ","image":null,"audio":null,"example":"彼の返事は曖昧だった。"},
		{"level":"n2","category":"vocabulary","name":"","hiragana":"テスト","mean":"invalid"}
	]`)
	if err != nil {
		t.Fatalf("import dataset: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("imported = %d, want 1", result.Imported)
	}
	if result.Skipped != 1 {
		t.Fatalf("skipped = %d, want 1", result.Skipped)
	}

	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Cards != 1 {
		t.Fatalf("cards = %d, want 1", stats.Cards)
	}
}
