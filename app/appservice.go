package main

import "time"

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
}

type AppService struct {
	startedAt time.Time
}

func NewAppService() *AppService {
	return &AppService{startedAt: time.Now().UTC()}
}

func (s *AppService) Status() AppStatus {
	return AppStatus{
		Name:         appDisplayName,
		Runtime:      "Golang + Wails " + wailsVersion,
		Frontend:     "Svelte " + svelteVersion + " + Vite",
		Storage:      "SQLite3 planned for Milestone 1",
		StartedAt:    s.startedAt.Format(time.RFC3339),
		ImportTarget: "User-uploaded JLPT JSON datasets",
	}
}

func (s *AppService) Echo(message string) string {
	if message == "" {
		return "Go service is reachable from Svelte."
	}

	return "Go service received: " + message
}
