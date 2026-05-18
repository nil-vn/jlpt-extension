package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nil-vn/jlpt-extension/app/internal/flashcards"
)

const (
	appDisplayName = "JLPT Flashcard Desktop"
	wailsVersion   = "v3.0.0-alpha.93"
	svelteVersion  = "5"
)

type AppStatus struct {
	Name         string `json:"name"`
	Runtime      string `json:"runtime"`
	Frontend     string `json:"frontend"`
	Storage      string `json:"storage"`
	StartedAt    string `json:"startedAt"`
	ImportTarget string `json:"importTarget"`
	LibraryCount int    `json:"libraryCount"`
}

type FlashcardDTO struct {
	ID       string  `json:"id"`
	Level    string  `json:"level"`
	Category string  `json:"category"`
	Name     string  `json:"name"`
	Mean     string  `json:"mean"`
	Hiragana string  `json:"hiragana"`
	Image    *string `json:"image"`
	Audio    *string `json:"audio"`
	Example  *string `json:"example"`
}

type FlashcardFilter struct {
	Level    string `json:"level"`
	Category string `json:"category"`
	Search   string `json:"search"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

type LibrarySummary struct {
	Total   int            `json:"total"`
	Shown   int            `json:"shown"`
	ByLevel map[string]int `json:"byLevel"`
}

type ImportOptions struct {
	ReplaceLibrary bool `json:"replaceLibrary"`
	DryRun         bool `json:"dryRun"`
}

type ValidationError struct {
	Index   *int   `json:"index,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type ImportResult struct {
	BatchID      int64             `json:"batchId"`
	SourceSHA256 string            `json:"sourceSha256"`
	TotalRows    int               `json:"totalRows"`
	ValidRows    int               `json:"validRows"`
	InvalidRows  int               `json:"invalidRows"`
	Inserted     int               `json:"inserted"`
	Updated      int               `json:"updated"`
	Errors       []ValidationError `json:"errors"`
}

type AppService struct {
	startedAt time.Time
	importer  *flashcards.ImportService
	repo      *flashcards.Repository
}

func NewAppService(db *sql.DB) *AppService {
	return &AppService{
		startedAt: time.Now().UTC(),
		importer:  flashcards.NewImportService(db),
		repo:      flashcards.NewRepository(db),
	}
}

func (s *AppService) Status() (AppStatus, error) {
	count, err := s.repo.Count(context.Background())
	if err != nil {
		return AppStatus{}, err
	}
	return AppStatus{
		Name:         appDisplayName,
		Runtime:      "Golang + Wails " + wailsVersion,
		Frontend:     "Svelte " + svelteVersion + " + Vite",
		Storage:      "SQLite3 library with JSON import service",
		StartedAt:    s.startedAt.Format(time.RFC3339),
		ImportTarget: "User-uploaded JLPT JSON datasets",
		LibraryCount: count,
	}, nil
}

func (s *AppService) Echo(message string) string {
	if message == "" {
		return "Go service is reachable from Svelte."
	}

	return "Go service received: " + message
}

func (s *AppService) PreviewImportJSON(filename string, content string) (ImportResult, error) {
	return s.importJSON(filename, content, ImportOptions{DryRun: true})
}

func (s *AppService) ImportFlashcardsFromJSON(filename string, content string, options ImportOptions) (ImportResult, error) {
	options.DryRun = false
	return s.importJSON(filename, content, options)
}

func (s *AppService) ListFlashcards(filter FlashcardFilter) ([]FlashcardDTO, error) {
	if err := validateLibraryFilter(filter); err != nil {
		return nil, err
	}
	cards, err := s.repo.List(context.Background(), toRepositoryFilter(filter))
	if err != nil {
		return nil, err
	}

	items := make([]FlashcardDTO, 0, len(cards))
	for _, card := range cards {
		items = append(items, toFlashcardDTO(card))
	}
	return items, nil
}

func (s *AppService) LibrarySummary() (LibrarySummary, error) {
	cards, err := s.repo.List(context.Background(), flashcards.Filter{})
	if err != nil {
		return LibrarySummary{}, err
	}

	byLevel := map[string]int{"n5": 0, "n4": 0, "n3": 0, "n2": 0, "n1": 0}
	for _, card := range cards {
		byLevel[string(card.Level)]++
	}
	return LibrarySummary{Total: len(cards), Shown: len(cards), ByLevel: byLevel}, nil
}

func (s *AppService) importJSON(filename string, content string, options ImportOptions) (ImportResult, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "uploaded-flashcards.json"
	}

	result, err := s.importer.ImportFlashcardsFromBytes(context.Background(), filename, []byte(content), flashcards.ImportOptions{
		ReplaceLibrary: options.ReplaceLibrary,
		DryRun:         options.DryRun,
	})
	if err != nil {
		return ImportResult{}, err
	}
	return toImportResult(result), nil
}

func toRepositoryFilter(filter FlashcardFilter) flashcards.Filter {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	repoFilter := flashcards.Filter{Search: strings.TrimSpace(filter.Search), Limit: limit, Offset: filter.Offset}
	if filter.Level != "" {
		level := flashcards.Level(filter.Level)
		repoFilter.Level = &level
	}
	if filter.Category != "" {
		category := flashcards.Category(filter.Category)
		repoFilter.Category = &category
	}
	return repoFilter
}

func toFlashcardDTO(card flashcards.Flashcard) FlashcardDTO {
	return FlashcardDTO{
		ID:       card.ID,
		Level:    string(card.Level),
		Category: string(card.Category),
		Name:     card.Name,
		Mean:     card.Mean,
		Hiragana: card.Hiragana,
		Image:    card.Image,
		Audio:    card.Audio,
		Example:  card.Example,
	}
}

func toImportResult(result flashcards.ImportResult) ImportResult {
	errs := make([]ValidationError, 0, len(result.Errors))
	for _, item := range result.Errors {
		errs = append(errs, ValidationError{Index: item.Index, Field: item.Field, Message: item.Message})
	}
	return ImportResult{
		BatchID:      result.BatchID,
		SourceSHA256: result.SourceSHA256,
		TotalRows:    result.TotalRows,
		ValidRows:    result.ValidRows,
		InvalidRows:  result.InvalidRows,
		Inserted:     result.Inserted,
		Updated:      result.Updated,
		Errors:       errs,
	}
}

func validateLibraryFilter(filter FlashcardFilter) error {
	if filter.Limit < 0 || filter.Offset < 0 {
		return fmt.Errorf("limit and offset must not be negative")
	}
	return nil
}
