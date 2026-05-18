package flashcards

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Card mirrors the Svelte flashcard shape so imported extension JSON can be
// persisted without changing the user's source data format.
type Card struct {
	Level    string  `json:"level"`
	Category string  `json:"category"`
	Name     string  `json:"name"`
	Mean     string  `json:"mean"`
	Hiragana string  `json:"hiragana"`
	Image    *string `json:"image"`
	Audio    *string `json:"audio"`
	Example  *string `json:"example"`
}

// ID returns a stable deterministic key equivalent in spirit to the extension
// createFlashcardId helper, with a hash to keep the SQLite primary key compact.
func (c Card) ID() string {
	parts := []string{c.Level, c.Category, c.Name, c.Hiragana, c.Mean}
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func (c Card) Validate() error {
	if strings.TrimSpace(c.Level) == "" {
		return fmt.Errorf("level is required")
	}
	if strings.TrimSpace(c.Category) == "" {
		return fmt.Errorf("category is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(c.Mean) == "" {
		return fmt.Errorf("mean is required")
	}
	if strings.TrimSpace(c.Hiragana) == "" {
		return fmt.Errorf("hiragana is required")
	}
	return nil
}
