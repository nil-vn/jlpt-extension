package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/nil-vn/jlpt-extension/app/internal/flashcards"
	"github.com/nil-vn/jlpt-extension/app/internal/settings"
	"github.com/nil-vn/jlpt-extension/app/internal/study"
)

const (
	appDisplayName = "JLPT Flashcard Desktop"
	wailsVersion   = "v3.0.0-alpha.93"
	svelteVersion  = "5"

	studyPreferencesKey = "study.preferences"
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
	ID         string  `json:"id"`
	Level      string  `json:"level"`
	Category   string  `json:"category"`
	Name       string  `json:"name"`
	Mean       string  `json:"mean"`
	Hiragana   string  `json:"hiragana"`
	Image      *string `json:"image"`
	Audio      *string `json:"audio"`
	Example    *string `json:"example"`
	Note       string  `json:"note"`
	Bookmarked bool    `json:"bookmarked"`
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

type StudySettingsDTO struct {
	SelectedLevels    []string `json:"selectedLevels"`
	EnabledCategories []string `json:"enabledCategories"`
	OrderMode         string   `json:"orderMode"`
	RevealAnswers     bool     `json:"revealAnswers"`
	DailyGoal         int      `json:"dailyGoal"`
}

type StudyStateDTO struct {
	CurrentCard  *FlashcardDTO    `json:"currentCard"`
	CurrentIndex int              `json:"currentIndex"`
	TotalCards   int              `json:"totalCards"`
	Settings     StudySettingsDTO `json:"settings"`
}

type AppService struct {
	startedAt    time.Time
	importer     *flashcards.ImportService
	repo         *flashcards.Repository
	studyRepo    *study.Repository
	settingsRepo *settings.Repository
}

func NewAppService(db *sql.DB) *AppService {
	return &AppService{
		startedAt:    time.Now().UTC(),
		importer:     flashcards.NewImportService(db),
		repo:         flashcards.NewRepository(db),
		studyRepo:    study.NewRepository(db),
		settingsRepo: settings.NewRepository(db),
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
		item, err := s.toFlashcardDTO(context.Background(), card)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
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

func (s *AppService) GetStudyState() (StudyStateDTO, error) {
	return s.studyState(context.Background(), "current")
}

func (s *AppService) MoveNext() (StudyStateDTO, error) {
	return s.studyState(context.Background(), "next")
}

func (s *AppService) MovePrevious() (StudyStateDTO, error) {
	return s.studyState(context.Background(), "previous")
}

func (s *AppService) UpdateStudySettings(input StudySettingsDTO) (StudyStateDTO, error) {
	ctx := context.Background()
	prefs := sanitizeStudySettings(input)
	if err := settings.Set(ctx, s.settingsRepo, studyPreferencesKey, prefs); err != nil {
		return StudyStateDTO{}, err
	}
	state, err := s.studyRepo.Get(ctx)
	if err != nil {
		return StudyStateDTO{}, err
	}
	state.OrderMode = study.OrderMode(prefs.OrderMode)
	state.RevealAnswers = prefs.RevealAnswers
	if err := s.studyRepo.Save(ctx, state); err != nil {
		return StudyStateDTO{}, err
	}
	return s.studyState(ctx, "current")
}

func (s *AppService) SaveNote(cardID string, note string) (FlashcardDTO, error) {
	ctx := context.Background()
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return FlashcardDTO{}, fmt.Errorf("cardID is required")
	}
	if err := s.repo.SaveNote(ctx, cardID, note); err != nil {
		return FlashcardDTO{}, err
	}
	card, found, err := s.repo.Get(ctx, cardID)
	if err != nil {
		return FlashcardDTO{}, err
	}
	if !found {
		return FlashcardDTO{}, fmt.Errorf("flashcard %q not found", cardID)
	}
	return s.toFlashcardDTO(ctx, card)
}

func (s *AppService) ToggleBookmark(cardID string) (FlashcardDTO, error) {
	ctx := context.Background()
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return FlashcardDTO{}, fmt.Errorf("cardID is required")
	}
	if _, err := s.repo.ToggleBookmark(ctx, cardID); err != nil {
		return FlashcardDTO{}, err
	}
	card, found, err := s.repo.Get(ctx, cardID)
	if err != nil {
		return FlashcardDTO{}, err
	}
	if !found {
		return FlashcardDTO{}, fmt.Errorf("flashcard %q not found", cardID)
	}
	return s.toFlashcardDTO(ctx, card)
}

func (s *AppService) studyState(ctx context.Context, direction string) (StudyStateDTO, error) {
	prefs, err := s.studySettings(ctx)
	if err != nil {
		return StudyStateDTO{}, err
	}
	state, err := s.studyRepo.Get(ctx)
	if err != nil {
		return StudyStateDTO{}, err
	}
	cards, err := s.studyCards(ctx, prefs)
	if err != nil {
		return StudyStateDTO{}, err
	}
	if len(cards) == 0 {
		state.CurrentCardID = nil
		state.OrderMode = study.OrderMode(prefs.OrderMode)
		state.RevealAnswers = prefs.RevealAnswers
		if err := s.studyRepo.Save(ctx, state); err != nil {
			return StudyStateDTO{}, err
		}
		return StudyStateDTO{CurrentIndex: -1, TotalCards: 0, Settings: prefs}, nil
	}

	currentIndex := findCardIndex(cards, state.CurrentCardID)
	if currentIndex < 0 {
		currentIndex = 0
	}
	switch direction {
	case "next":
		if prefs.OrderMode == string(study.OrderModeRandom) && len(cards) > 1 {
			next := rand.Intn(len(cards) - 1)
			if next >= currentIndex {
				next++
			}
			currentIndex = next
		} else {
			currentIndex = (currentIndex + 1) % len(cards)
		}
	case "previous":
		currentIndex = (currentIndex - 1 + len(cards)) % len(cards)
	}

	state.CurrentCardID = &cards[currentIndex].ID
	state.OrderMode = study.OrderMode(prefs.OrderMode)
	state.RevealAnswers = prefs.RevealAnswers
	if err := s.studyRepo.Save(ctx, state); err != nil {
		return StudyStateDTO{}, err
	}
	current, err := s.toFlashcardDTO(ctx, cards[currentIndex])
	if err != nil {
		return StudyStateDTO{}, err
	}
	return StudyStateDTO{CurrentCard: &current, CurrentIndex: currentIndex, TotalCards: len(cards), Settings: prefs}, nil
}

func (s *AppService) studySettings(ctx context.Context) (StudySettingsDTO, error) {
	prefs, found, err := settings.Get[StudySettingsDTO](ctx, s.settingsRepo, studyPreferencesKey)
	if err != nil {
		return StudySettingsDTO{}, err
	}
	if !found {
		prefs = defaultStudySettings()
		if err := settings.Set(ctx, s.settingsRepo, studyPreferencesKey, prefs); err != nil {
			return StudySettingsDTO{}, err
		}
	}
	prefs = sanitizeStudySettings(prefs)
	state, err := s.studyRepo.Get(ctx)
	if err != nil {
		return StudySettingsDTO{}, err
	}
	prefs.OrderMode = string(state.OrderMode)
	prefs.RevealAnswers = state.RevealAnswers
	if prefs.OrderMode != string(study.OrderModeRandom) {
		prefs.OrderMode = string(study.OrderModeSequential)
	}
	return prefs, nil
}

func (s *AppService) studyCards(ctx context.Context, prefs StudySettingsDTO) ([]flashcards.Flashcard, error) {
	cards, err := s.repo.List(ctx, flashcards.Filter{})
	if err != nil {
		return nil, err
	}
	levelSet := stringSet(prefs.SelectedLevels)
	categorySet := stringSet(prefs.EnabledCategories)
	filtered := make([]flashcards.Flashcard, 0, len(cards))
	for _, card := range cards {
		if levelSet[string(card.Level)] && categorySet[string(card.Category)] {
			filtered = append(filtered, card)
		}
	}
	return filtered, nil
}

func (s *AppService) toFlashcardDTO(ctx context.Context, card flashcards.Flashcard) (FlashcardDTO, error) {
	note, err := s.repo.GetNote(ctx, card.ID)
	if err != nil {
		return FlashcardDTO{}, err
	}
	bookmarked, err := s.repo.IsBookmarked(ctx, card.ID)
	if err != nil {
		return FlashcardDTO{}, err
	}
	return FlashcardDTO{
		ID:         card.ID,
		Level:      string(card.Level),
		Category:   string(card.Category),
		Name:       card.Name,
		Mean:       card.Mean,
		Hiragana:   card.Hiragana,
		Image:      card.Image,
		Audio:      card.Audio,
		Example:    card.Example,
		Note:       note,
		Bookmarked: bookmarked,
	}, nil
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

func defaultStudySettings() StudySettingsDTO {
	return StudySettingsDTO{
		SelectedLevels:    []string{"n5", "n4", "n3", "n2", "n1"},
		EnabledCategories: []string{"gramma", "vocabulary", "kanji", "reading", "listening"},
		OrderMode:         string(study.OrderModeSequential),
		RevealAnswers:     false,
		DailyGoal:         10,
	}
}

func sanitizeStudySettings(input StudySettingsDTO) StudySettingsDTO {
	defaults := defaultStudySettings()
	prefs := StudySettingsDTO{
		SelectedLevels:    normalizeSelection(input.SelectedLevels, defaults.SelectedLevels, map[string]bool{"n5": true, "n4": true, "n3": true, "n2": true, "n1": true}),
		EnabledCategories: normalizeSelection(input.EnabledCategories, defaults.EnabledCategories, map[string]bool{"gramma": true, "vocabulary": true, "kanji": true, "reading": true, "listening": true}),
		OrderMode:         input.OrderMode,
		RevealAnswers:     input.RevealAnswers,
		DailyGoal:         input.DailyGoal,
	}
	if prefs.OrderMode != string(study.OrderModeRandom) {
		prefs.OrderMode = string(study.OrderModeSequential)
	}
	if prefs.DailyGoal <= 0 {
		prefs.DailyGoal = defaults.DailyGoal
	}
	return prefs
}

func normalizeSelection(values []string, fallback []string, allowed map[string]bool) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if allowed[value] && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}

func findCardIndex(cards []flashcards.Flashcard, currentID *string) int {
	if currentID == nil {
		return -1
	}
	for index, card := range cards {
		if card.ID == *currentID {
			return index
		}
	}
	return -1
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
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
