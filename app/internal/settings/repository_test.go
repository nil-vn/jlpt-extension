package settings_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
	"github.com/nil-vn/jlpt-extension/app/internal/settings"
)

func TestRepositorySetAndGetTypedJSON(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()
	repo := settings.NewRepository(db)

	type preferences struct {
		Theme     string   `json:"theme"`
		Levels    []string `json:"levels"`
		DailyGoal int      `json:"dailyGoal"`
	}
	want := preferences{Theme: "dark", Levels: []string{"n5", "n4"}, DailyGoal: 20}
	if err := settings.Set(ctx, repo, "study.preferences", want); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, found, err := settings.Get[preferences](ctx, repo, "study.preferences")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() found = false, want true")
	}
	if got.Theme != want.Theme || got.DailyGoal != want.DailyGoal || len(got.Levels) != len(want.Levels) {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}

	if err := repo.SetRaw(ctx, "bad", json.RawMessage(`{`)); err == nil {
		t.Fatal("SetRaw() invalid JSON error = nil, want error")
	}
}
