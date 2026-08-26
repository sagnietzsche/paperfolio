package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSlugProducesSafeReadableNames(t *testing.T) {
	cases := map[string]string{
		"Attention Is All You Need":      "attention-is-all-you-need",
		"  A title / with punctuation! ": "a-title-with-punctuation",
		"":                               "untitled",
	}
	for input, want := range cases {
		if got := slug(input); got != want {
			t.Errorf("slug(%q) = %q, want %q", input, got, want)
		}
		if strings.ContainsAny(slug(input), `/\\`) {
			t.Errorf("slug(%q) contains a path separator", input)
		}
	}
}

func TestStorePersistsAnnotationsAndMirrorsNotes(t *testing.T) {
	store := testStore(t)
	year := 2024
	paper, err := store.CreatePaper(PaperDraft{Title: "A useful paper", Authors: "Ada Lovelace", Year: &year}, bytes.NewReader([]byte("%PDF-1.7 test")))
	if err != nil {
		t.Fatal(err)
	}
	if paper.PDFPath == nil {
		t.Fatal("expected PDF path")
	}
	pdfPath := filepath.Join(store.UploadsDir, *paper.PDFPath)
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("imported PDF missing: %v", err)
	}

	page := 3
	highlight, err := store.CreateHighlight(Highlight{PaperID: paper.ID, Content: "Important result", Position: `{"pageNumber":3}`, Page: &page})
	if err != nil {
		t.Fatal(err)
	}
	if highlight.ID == "" {
		t.Fatal("expected highlight id")
	}
	note, err := store.CreateNote(paper.ID, "Reading notes", "## Result\n\nKeep this.")
	if err != nil {
		t.Fatal(err)
	}
	idea, err := store.CreateIdea(paper.ID, "Try this on another dataset")
	if err != nil {
		t.Fatal(err)
	}
	if idea.ID == "" {
		t.Fatal("expected idea id")
	}

	noteDir := paperNotesDir(store.DataDir, paper.Title, paper.ID)
	files, err := os.ReadDir(noteDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("expected one mirrored note: files=%d err=%v", len(files), err)
	}
	mirrored, err := os.ReadFile(filepath.Join(noteDir, files[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mirrored), "title: Reading notes") || !strings.Contains(string(mirrored), "Keep this.") {
		t.Fatalf("unexpected markdown mirror: %s", mirrored)
	}

	newTitle := "A renamed paper"
	if _, err = store.UpdatePaper(paper.ID, PaperPatch{Title: &newTitle}); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(paperNotesDir(store.DataDir, newTitle, paper.ID)); err != nil {
		t.Fatalf("notes directory was not renamed: %v", err)
	}
	updatedContent := "changed"
	if _, err = store.UpdateNote(note.ID, "Renamed note", updatedContent); err != nil {
		t.Fatal(err)
	}
	updatedHighlight, err := store.UpdateHighlight(highlight.ID, stringPtr("A comment"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if updatedHighlight.Comment == nil || *updatedHighlight.Comment != "A comment" {
		t.Fatalf("highlight note was not persisted: %#v", updatedHighlight.Comment)
	}
	storedHighlights, err := store.ListHighlights(paper.ID)
	if err != nil || len(storedHighlights) != 1 || storedHighlights[0].Page == nil || *storedHighlights[0].Page != 3 {
		t.Fatalf("highlight page data was not persisted: %#v, %v", storedHighlights, err)
	}

	if err = store.DeletePaper(paper.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetPaper(paper.ID); err == nil {
		t.Fatal("deleted paper still exists")
	}
	if _, err = store.ListHighlights(paper.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(pdfPath); !os.IsNotExist(err) {
		t.Fatalf("PDF was not removed: %v", err)
	}
	if _, err = os.Stat(paperNotesDir(store.DataDir, newTitle, paper.ID)); !os.IsNotExist(err) {
		t.Fatalf("note directory was not removed: %v", err)
	}
}

func TestStoreValidationAndClearing(t *testing.T) {
	store := testStore(t)
	if _, err := store.CreatePaper(PaperDraft{}, nil); err == nil {
		t.Fatal("expected title validation")
	}
	paper, err := store.CreatePaper(PaperDraft{Title: "Paper", Authors: "Author", Venue: "Venue"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	blank := ""
	updated, err := store.UpdatePaper(paper.ID, PaperPatch{Authors: &blank, Venue: &blank})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Authors != nil || updated.Venue != nil {
		t.Fatal("blank metadata should clear fields")
	}
	badStatus := "finished"
	if _, err = store.UpdatePaper(paper.ID, PaperPatch{Status: &badStatus}); err == nil {
		t.Fatal("expected status validation")
	}
	if _, err = store.CreateNote(paper.ID, "", "body"); err == nil {
		t.Fatal("expected note title validation")
	}
	if _, err = store.CreateIdea(paper.ID, " "); err == nil {
		t.Fatal("expected idea validation")
	}
}

func stringPtr(value string) *string { return &value }
