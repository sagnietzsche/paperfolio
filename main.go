package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"
)

const dataFolder = "Paperfolio_Data"

func dataDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, "Documents", dataFolder), nil
}

func main() {
	dataDir, err := dataDirectory()
	if err != nil {
		panic(err)
	}
	store, err := NewStore(dataDir)
	if err != nil {
		panic(err)
	}
	a := app.NewWithID("com.paperfolio.app")
	paperfolio := newPaperfolioApp(a, store)
	paperfolio.showWelcome()
	paperfolio.win.ShowAndRun()
}
