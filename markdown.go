package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// slug creates a readable, ASCII-only filesystem name without allowing titles
// to create path separators or unbounded filenames.
func slug(value string) string {
	var out strings.Builder
	pendingDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if pendingDash && out.Len() > 0 {
				out.WriteByte('-')
			}
			pendingDash = false
			if r < 128 {
				out.WriteRune(unicode.ToLower(r))
			} else {
				out.WriteString("x")
			}
		} else {
			pendingDash = true
		}
		if out.Len() >= 60 {
			break
		}
	}
	if out.Len() == 0 {
		return "untitled"
	}
	return out.String()
}

func stamped(title, id string) string {
	short := id
	if len(short) > 8 {
		short = short[:8]
	}
	return fmt.Sprintf("%s-%s", slug(title), short)
}

func paperNotesDir(dataDir, title, id string) string {
	return filepath.Join(dataDir, "notes", stamped(title, id))
}

func noteFile(dir, title, id string) string {
	return filepath.Join(dir, stamped(title, id)+".md")
}

func writeMarkdownNote(dataDir string, note Note, paper Paper) error {
	dir := paperNotesDir(dataDir, paper.Title, paper.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := noteFile(dir, note.Title, note.ID)
	removeOtherNoteFiles(dir, note.ID, target)
	content := ""
	if note.Content != nil {
		content = strings.TrimRight(*note.Content, "\r\n")
	}
	body := fmt.Sprintf("---\ntitle: %s\npaper: %s\nid: %s\nupdated: %s\n---\n\n%s\n", note.Title, paper.Title, note.ID, note.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"), content)
	return os.WriteFile(target, []byte(body), 0o644)
}

func removeOtherNoteFiles(dir, noteID, keep string) {
	short := noteID
	if len(short) > 8 {
		short = short[:8]
	}
	suffix := "-" + short + ".md"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if path != keep && strings.HasSuffix(entry.Name(), suffix) {
			_ = os.Remove(path)
		}
	}
}

func renamePaperDir(dataDir, oldTitle, newTitle, id string) error {
	from, to := paperNotesDir(dataDir, oldTitle, id), paperNotesDir(dataDir, newTitle, id)
	if from == to {
		return nil
	}
	if _, err := os.Stat(from); err != nil {
		return nil
	}
	if _, err := os.Stat(to); err == nil {
		return nil
	}
	return os.Rename(from, to)
}

func removePaperDir(dataDir, title, id string) { _ = os.RemoveAll(paperNotesDir(dataDir, title, id)) }

func (s *Store) mirrorNote(note Note) {
	paper, err := s.GetPaper(note.PaperID)
	if err != nil {
		return
	}
	_ = writeMarkdownNote(s.DataDir, note, *paper)
}

func (s *Store) removeNoteFiles(paperID, noteID, title string) {
	paper, err := s.GetPaper(paperID)
	if err != nil {
		return
	}
	dir := paperNotesDir(s.DataDir, paper.Title, paper.ID)
	removeOtherNoteFiles(dir, noteID, "")
	_ = os.Remove(noteFile(dir, title, noteID))
	_ = os.Remove(dir)
}
