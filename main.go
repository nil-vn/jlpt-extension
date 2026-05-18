package main

import (
	"embed"
	"log"

	"jlpt-extension/internal/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:dist
var assets embed.FS

func main() {
	storeService := storage.NewStoreService()
	databasePath, err := storage.DefaultDatabasePath()
	if err != nil {
		log.Fatal(err)
	}
	if err := storeService.Open(databasePath); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = storeService.Close() }()

	app := application.New(application.Options{
		Name:        "JLPT Study Companion",
		Description: "A Windows desktop JLPT flashcard companion built with Wails v3 and Svelte 5.",
		Services: []application.Service{
			application.NewService(storeService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "JLPT Study Companion",
		Width:     1080,
		Height:    760,
		MinWidth:  860,
		MinHeight: 640,
		URL:       "http://wails.localhost/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
