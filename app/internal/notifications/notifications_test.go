package notifications

import (
	"testing"
	"time"
)

func TestFormatPayloadMatchesExtensionContent(t *testing.T) {
	example := "日本語の例文"
	payload := FormatPayload(Flashcard{Level: "n2", Category: "vocabulary", Name: "試験", Mean: "exam", Hiragana: "しけん", Example: &example}, time.Unix(1700000000, 0).UTC())
	if payload.Title != "[N2 - Vocabulary]  試験 (しけん)" {
		t.Fatalf("Title = %q", payload.Title)
	}
	if payload.Message != "exam\n日本語の例文" {
		t.Fatalf("Message = %q", payload.Message)
	}
}

func TestIntervalDurationHasExtensionMinimum(t *testing.T) {
	if got := IntervalDuration(0.1); got != 30*time.Second {
		t.Fatalf("IntervalDuration(0.1) = %s, want 30s", got)
	}
	if got := IntervalDuration(3); got != 3*time.Minute {
		t.Fatalf("IntervalDuration(3) = %s, want 3m", got)
	}
}
