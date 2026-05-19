package notifications

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Flashcard struct {
	ID       string
	Level    string
	Category string
	Name     string
	Mean     string
	Hiragana string
	Example  *string
}

type Payload struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Card      Flashcard `json:"card"`
	CreatedAt string    `json:"createdAt"`
}

type Settings struct {
	Enabled         bool    `json:"enabled"`
	Paused          bool    `json:"paused"`
	IntervalMinutes float64 `json:"intervalMinutes"`
}

type Notifier interface {
	ShowFlashcard(ctx context.Context, payload Payload) error
}

type Provider interface {
	NextNotificationCard(ctx context.Context) (Flashcard, bool, error)
	NotificationSettings(ctx context.Context) (Settings, error)
}

type Scheduler struct {
	provider Provider
	notifier Notifier
	clock    func() time.Time

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewScheduler(provider Provider, notifier Notifier) *Scheduler {
	return &Scheduler{provider: provider, notifier: notifier, clock: func() time.Time { return time.Now().UTC() }}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.loop(runCtx, s.done)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (s *Scheduler) ShowNow(ctx context.Context) (Payload, bool, error) {
	card, found, err := s.provider.NextNotificationCard(ctx)
	if err != nil || !found {
		return Payload{}, found, err
	}
	payload := FormatPayload(card, s.clock())
	if err := s.notifier.ShowFlashcard(ctx, payload); err != nil {
		return Payload{}, true, err
	}
	return payload, true, nil
}

func (s *Scheduler) loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	nextDelay := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(nextDelay):
		}

		settings, err := s.provider.NotificationSettings(ctx)
		if err != nil || !settings.Enabled || settings.Paused {
			nextDelay = time.Second
			continue
		}
		if _, _, err := s.ShowNow(ctx); err != nil {
			nextDelay = time.Second
			continue
		}
		nextDelay = IntervalDuration(settings.IntervalMinutes)
	}
}

func IntervalDuration(minutes float64) time.Duration {
	if minutes < 0.5 {
		minutes = 0.5
	}
	return time.Duration(minutes * float64(time.Minute))
}

func FormatPayload(card Flashcard, now time.Time) Payload {
	message := card.Mean
	if card.Example != nil && *card.Example != "" {
		message += "\n" + *card.Example
	}
	return Payload{
		ID:        fmt.Sprintf("jlpt-card-reminder-%d-%d", now.UnixMilli(), rand.Intn(100000)),
		Title:     FormatTitle(card),
		Message:   message,
		Card:      card,
		CreatedAt: now.Format(time.RFC3339),
	}
}

func FormatTitle(card Flashcard) string {
	reading := ""
	if card.Hiragana != "" {
		reading = " (" + card.Hiragana + ")"
	}
	return fmt.Sprintf("[%s - %s]  %s%s", upperASCII(card.Level), categoryLabel(card.Category), card.Name, reading)
}

func categoryLabel(category string) string {
	switch category {
	case "gramma":
		return "Grammar"
	case "vocabulary":
		return "Vocabulary"
	case "kanji":
		return "Kanji"
	case "reading":
		return "Reading"
	case "listening":
		return "Listening"
	default:
		return category
	}
}

func upperASCII(value string) string {
	out := []byte(value)
	for i, b := range out {
		if b >= 'a' && b <= 'z' {
			out[i] = b - ('a' - 'A')
		}
	}
	return string(out)
}
