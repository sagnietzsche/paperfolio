package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS papers (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  authors TEXT,
  abstract TEXT,
  year INTEGER,
  venue TEXT,
  url TEXT,
  pdf_path TEXT,
  status TEXT NOT NULL DEFAULT 'unread',
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE TABLE IF NOT EXISTS highlights (
  id TEXT PRIMARY KEY,
  paper_id TEXT NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  comment TEXT,
  emoji TEXT,
  position TEXT NOT NULL,
  page INTEGER,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  paper_id TEXT NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  content TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE TABLE IF NOT EXISTS ideas (
  id TEXT PRIMARY KEY,
  paper_id TEXT NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_highlights_paper ON highlights(paper_id);
CREATE INDEX IF NOT EXISTS idx_notes_paper ON notes(paper_id);
CREATE INDEX IF NOT EXISTS idx_ideas_paper ON ideas(paper_id);
CREATE TABLE IF NOT EXISTS latex_documents (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  template TEXT NOT NULL DEFAULT 'article',
  paper_id TEXT REFERENCES papers(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_latex_docs_paper ON latex_documents(paper_id);
`

type Store struct {
	db         *sql.DB
	DataDir    string
	UploadsDir string
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "uploads"), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite3", filepath.Join(dataDir, "paperfolio.db"))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return &Store{db: db, DataDir: dataDir, UploadsDir: filepath.Join(dataDir, "uploads")}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func cleanText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func scanPaper(scanner interface{ Scan(...any) error }, withCounts bool) (*Paper, error) {
	var p Paper
	var authors, abstract, venue, url, pdf, created, updated sql.NullString
	var year sql.NullInt64
	if withCounts {
		err := scanner.Scan(&p.ID, &p.Title, &authors, &abstract, &year, &venue, &url, &pdf, &p.Status, &created, &updated, &p.HighlightCount, &p.NoteCount, &p.IdeaCount)
		if err != nil {
			return nil, err
		}
	} else {
		if err := scanner.Scan(&p.ID, &p.Title, &authors, &abstract, &year, &venue, &url, &pdf, &p.Status, &created, &updated); err != nil {
			return nil, err
		}
	}
	p.Authors, p.Abstract, p.Venue, p.URL, p.PDFPath = nullString(authors), nullString(abstract), nullString(venue), nullString(url), nullString(pdf)
	if year.Valid {
		value := int(year.Int64)
		p.Year = &value
	}
	p.CreatedAt, p.UpdatedAt = parseTime(created.String), parseTime(updated.String)
	return &p, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

const paperColumns = `id, title, authors, abstract, year, venue, url, pdf_path, status, created_at, updated_at`

func (s *Store) ListPapers() ([]*Paper, error) {
	rows, err := s.db.Query(`SELECT ` + paperColumns + `,
		(SELECT count(*) FROM highlights h WHERE h.paper_id = p.id),
		(SELECT count(*) FROM notes n WHERE n.paper_id = p.id),
		(SELECT count(*) FROM ideas i WHERE i.paper_id = p.id)
		FROM papers p ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list papers: %w", err)
	}
	defer rows.Close()
	var papers []*Paper
	for rows.Next() {
		paper, scanErr := scanPaper(rows, true)
		if scanErr != nil {
			return nil, fmt.Errorf("read paper: %w", scanErr)
		}
		papers = append(papers, paper)
	}
	return papers, rows.Err()
}

func (s *Store) GetPaper(id string) (*Paper, error) {
	paper, err := scanPaper(s.db.QueryRow(`SELECT `+paperColumns+` FROM papers WHERE id = ?`, id), false)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("paper not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get paper: %w", err)
	}
	return paper, nil
}

func validateStatus(status string) error {
	if status != StatusUnread && status != StatusReading && status != StatusRead {
		return fmt.Errorf("invalid reading status %q", status)
	}
	return nil
}

func (s *Store) CreatePaper(draft PaperDraft, pdf io.Reader) (*Paper, error) {
	title := strings.TrimSpace(draft.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	var pdfName *string
	if pdf != nil {
		name, copyErr := s.savePDF(pdf)
		if copyErr != nil {
			return nil, copyErr
		}
		pdfName = &name
	}
	_, err = s.db.Exec(`INSERT INTO papers (id, title, authors, abstract, year, venue, url, pdf_path) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, title, cleanText(draft.Authors), cleanText(draft.Abstract), draft.Year, cleanText(draft.Venue), cleanText(draft.URL), pdfName)
	if err != nil {
		if pdfName != nil {
			_ = os.Remove(filepath.Join(s.UploadsDir, *pdfName))
		}
		return nil, fmt.Errorf("create paper: %w", err)
	}
	return s.GetPaper(id)
}

func (s *Store) savePDF(source io.Reader) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	name := id + ".pdf"
	file, err := os.OpenFile(filepath.Join(s.UploadsDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create PDF copy: %w", err)
	}
	_, copyErr := io.Copy(file, source)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("copy PDF: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("close PDF: %w", closeErr)
	}
	return name, nil
}

func (s *Store) SetPaperPDF(id string, pdf io.Reader) (*Paper, error) {
	old, err := s.GetPaper(id)
	if err != nil {
		return nil, err
	}
	name, err := s.savePDF(pdf)
	if err != nil {
		return nil, err
	}
	if _, err = s.db.Exec(`UPDATE papers SET pdf_path = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, name, id); err != nil {
		_ = os.Remove(filepath.Join(s.UploadsDir, name))
		return nil, fmt.Errorf("replace PDF: %w", err)
	}
	if old.PDFPath != nil {
		_ = os.Remove(filepath.Join(s.UploadsDir, filepath.Base(*old.PDFPath)))
	}
	return s.GetPaper(id)
}

func (s *Store) UpdatePaper(id string, patch PaperPatch) (*Paper, error) {
	old, err := s.GetPaper(id)
	if err != nil {
		return nil, err
	}
	sets, args := make([]string, 0, 7), make([]any, 0, 8)
	if patch.Title != nil {
		value := strings.TrimSpace(*patch.Title)
		if value == "" {
			return nil, fmt.Errorf("title is required")
		}
		sets, args = append(sets, "title = ?"), append(args, value)
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{"authors", patch.Authors}, {"abstract", patch.Abstract}, {"venue", patch.Venue}, {"url", patch.URL},
	} {
		if field.value != nil {
			sets, args = append(sets, field.name+" = ?"), append(args, cleanText(*field.value))
		}
	}
	if patch.Year != nil {
		sets, args = append(sets, "year = ?"), append(args, *patch.Year)
	}
	if patch.Status != nil {
		if err := validateStatus(*patch.Status); err != nil {
			return nil, err
		}
		sets, args = append(sets, "status = ?"), append(args, *patch.Status)
	}
	if len(sets) == 0 {
		return old, nil
	}
	sets = append(sets, "updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')")
	args = append(args, id)
	if _, err := s.db.Exec(`UPDATE papers SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
		return nil, fmt.Errorf("update paper: %w", err)
	}
	updated, err := s.GetPaper(id)
	if err == nil && old.Title != updated.Title {
		_ = renamePaperDir(s.DataDir, old.Title, updated.Title, id)
	}
	return updated, err
}

func (s *Store) DeletePaper(id string) error {
	paper, err := s.GetPaper(id)
	if err != nil {
		return err
	}
	if _, err = s.db.Exec(`DELETE FROM papers WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete paper: %w", err)
	}
	if paper.PDFPath != nil {
		_ = os.Remove(filepath.Join(s.UploadsDir, filepath.Base(*paper.PDFPath)))
	}
	removePaperDir(s.DataDir, paper.Title, paper.ID)
	return nil
}

func (s *Store) ListHighlights(paperID string) ([]Highlight, error) {
	rows, err := s.db.Query(`SELECT id, paper_id, content, comment, emoji, position, page, created_at FROM highlights WHERE paper_id = ? ORDER BY created_at ASC`, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Highlight
	for rows.Next() {
		var h Highlight
		var comment, emoji, position, created sql.NullString
		var page sql.NullInt64
		if err := rows.Scan(&h.ID, &h.PaperID, &h.Content, &comment, &emoji, &position, &page, &created); err != nil {
			return nil, err
		}
		h.Comment, h.Emoji, h.Position, h.CreatedAt = nullString(comment), nullString(emoji), position.String, parseTime(created.String)
		if page.Valid {
			value := int(page.Int64)
			h.Page = &value
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

func (s *Store) CreateHighlight(h Highlight) (Highlight, error) {
	if strings.TrimSpace(h.Content) == "" || strings.TrimSpace(h.Position) == "" {
		return Highlight{}, fmt.Errorf("highlight content and position are required")
	}
	if h.ID == "" {
		var err error
		h.ID, err = newID()
		if err != nil {
			return Highlight{}, err
		}
	}
	_, err := s.db.Exec(`INSERT INTO highlights (id, paper_id, content, comment, emoji, position, page) VALUES (?, ?, ?, ?, ?, ?, ?)`, h.ID, h.PaperID, strings.TrimSpace(h.Content), cleanText(valueOrEmpty(h.Comment)), cleanText(valueOrEmpty(h.Emoji)), h.Position, h.Page)
	if err != nil {
		return Highlight{}, fmt.Errorf("create highlight: %w", err)
	}
	return s.getHighlight(h.ID)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Store) getHighlight(id string) (Highlight, error) {
	var h Highlight
	var comment, emoji, created sql.NullString
	var page sql.NullInt64
	err := s.db.QueryRow(`SELECT id, paper_id, content, comment, emoji, position, page, created_at FROM highlights WHERE id = ?`, id).Scan(&h.ID, &h.PaperID, &h.Content, &comment, &emoji, &h.Position, &page, &created)
	if err != nil {
		return Highlight{}, err
	}
	h.Comment, h.Emoji, h.CreatedAt = nullString(comment), nullString(emoji), parseTime(created.String)
	if page.Valid {
		value := int(page.Int64)
		h.Page = &value
	}
	return h, nil
}

func (s *Store) UpdateHighlight(id string, comment, emoji *string) (Highlight, error) {
	if _, err := s.db.Exec(`UPDATE highlights SET comment = ?, emoji = ? WHERE id = ?`, comment, emoji, id); err != nil {
		return Highlight{}, err
	}
	return s.getHighlight(id)
}

func (s *Store) DeleteHighlight(id string) error {
	_, err := s.db.Exec(`DELETE FROM highlights WHERE id = ?`, id)
	return err
}

func (s *Store) ListNotes(paperID string) ([]Note, error) {
	rows, err := s.db.Query(`SELECT id, paper_id, title, content, created_at, updated_at FROM notes WHERE paper_id = ? ORDER BY updated_at DESC`, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, note)
	}
	return result, rows.Err()
}

func scanNote(scanner interface{ Scan(...any) error }) (Note, error) {
	var n Note
	var content, created, updated sql.NullString
	err := scanner.Scan(&n.ID, &n.PaperID, &n.Title, &content, &created, &updated)
	n.Content = nullString(content)
	n.CreatedAt, n.UpdatedAt = parseTime(created.String), parseTime(updated.String)
	return n, err
}

func (s *Store) CreateNote(paperID, title, content string) (Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Note{}, fmt.Errorf("note title is required")
	}
	id, err := newID()
	if err != nil {
		return Note{}, err
	}
	if _, err = s.db.Exec(`INSERT INTO notes (id, paper_id, title, content) VALUES (?, ?, ?, ?)`, id, paperID, title, cleanText(content)); err != nil {
		return Note{}, fmt.Errorf("create note: %w", err)
	}
	note, err := s.getNote(id)
	if err == nil {
		s.mirrorNote(note)
	}
	return note, err
}

func (s *Store) getNote(id string) (Note, error) {
	return scanNote(s.db.QueryRow(`SELECT id, paper_id, title, content, created_at, updated_at FROM notes WHERE id = ?`, id))
}

func (s *Store) UpdateNote(id, title, content string) (Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Note{}, fmt.Errorf("note title is required")
	}
	if _, err := s.db.Exec(`UPDATE notes SET title = ?, content = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, title, cleanText(content), id); err != nil {
		return Note{}, err
	}
	note, err := s.getNote(id)
	if err == nil {
		s.mirrorNote(note)
	}
	return note, err
}

func (s *Store) DeleteNote(id string) error {
	var paperID, title string
	err := s.db.QueryRow(`SELECT paper_id, title FROM notes WHERE id = ?`, id).Scan(&paperID, &title)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = s.db.Exec(`DELETE FROM notes WHERE id = ?`, id); err != nil {
		return err
	}
	if err == nil {
		s.removeNoteFiles(paperID, id, title)
	}
	return nil
}

func (s *Store) ListIdeas(paperID string) ([]Idea, error) {
	rows, err := s.db.Query(`SELECT id, paper_id, content, created_at FROM ideas WHERE paper_id = ? ORDER BY created_at DESC`, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Idea
	for rows.Next() {
		var idea Idea
		var created string
		if err := rows.Scan(&idea.ID, &idea.PaperID, &idea.Content, &created); err != nil {
			return nil, err
		}
		idea.CreatedAt = parseTime(created)
		result = append(result, idea)
	}
	return result, rows.Err()
}

func (s *Store) CreateIdea(paperID, content string) (Idea, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Idea{}, fmt.Errorf("idea content is required")
	}
	id, err := newID()
	if err != nil {
		return Idea{}, err
	}
	if _, err = s.db.Exec(`INSERT INTO ideas (id, paper_id, content) VALUES (?, ?, ?)`, id, paperID, content); err != nil {
		return Idea{}, err
	}
	var idea Idea
	var created string
	err = s.db.QueryRow(`SELECT id, paper_id, content, created_at FROM ideas WHERE id = ?`, id).Scan(&idea.ID, &idea.PaperID, &idea.Content, &created)
	idea.CreatedAt = parseTime(created)
	return idea, err
}

func (s *Store) DeleteIdea(id string) error {
	_, err := s.db.Exec(`DELETE FROM ideas WHERE id = ?`, id)
	return err
}

func scanLatexDocument(scanner interface{ Scan(...any) error }) (LatexDocument, error) {
	var d LatexDocument
	var paperID sql.NullString
	var created, updated string
	err := scanner.Scan(&d.ID, &d.Title, &d.Content, &d.Template, &paperID, &created, &updated)
	if err != nil {
		return LatexDocument{}, err
	}
	if paperID.Valid {
		d.PaperID = &paperID.String
	}
	d.CreatedAt = parseTime(created)
	d.UpdatedAt = parseTime(updated)
	return d, nil
}

func (s *Store) ListLatexDocuments() ([]LatexDocument, error) {
	rows, err := s.db.Query(`SELECT id, title, content, template, paper_id, created_at, updated_at FROM latex_documents ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []LatexDocument
	for rows.Next() {
		doc, err := scanLatexDocument(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, doc)
	}
	return result, rows.Err()
}

func (s *Store) ListLatexDocumentsByPaper(paperID string) ([]LatexDocument, error) {
	rows, err := s.db.Query(`SELECT id, title, content, template, paper_id, created_at, updated_at FROM latex_documents WHERE paper_id = ? ORDER BY updated_at DESC`, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []LatexDocument
	for rows.Next() {
		doc, err := scanLatexDocument(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, doc)
	}
	return result, rows.Err()
}

func (s *Store) GetLatexDocument(id string) (LatexDocument, error) {
	doc, err := scanLatexDocument(s.db.QueryRow(`SELECT id, title, content, template, paper_id, created_at, updated_at FROM latex_documents WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return LatexDocument{}, fmt.Errorf("latex document not found")
	}
	return doc, err
}

func (s *Store) CreateLatexDocument(title, content, template string, paperID *string) (LatexDocument, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return LatexDocument{}, fmt.Errorf("latex document title is required")
	}
	if template == "" {
		template = "article"
	}
	if content == "" {
		content = latexTemplateContent(template, title)
	}
	id, err := newID()
	if err != nil {
		return LatexDocument{}, err
	}
	var pid any
	if paperID != nil && strings.TrimSpace(*paperID) != "" {
		v := strings.TrimSpace(*paperID)
		pid = v
		paperID = &v
	}
	if _, err = s.db.Exec(`INSERT INTO latex_documents (id, title, content, template, paper_id) VALUES (?, ?, ?, ?, ?)`, id, title, content, template, pid); err != nil {
		return LatexDocument{}, fmt.Errorf("create latex document: %w", err)
	}
	doc, err := s.GetLatexDocument(id)
	if err == nil {
		s.mirrorLatexDocument(doc)
	}
	return doc, err
}

func (s *Store) UpdateLatexDocument(id, title, content, template string, paperID *string) (LatexDocument, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return LatexDocument{}, fmt.Errorf("latex document title is required")
	}
	if template == "" {
		template = "article"
	}
	var pid any
	if paperID != nil && strings.TrimSpace(*paperID) != "" {
		v := strings.TrimSpace(*paperID)
		pid = v
	}
	if _, err := s.db.Exec(`UPDATE latex_documents SET title = ?, content = ?, template = ?, paper_id = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, title, content, template, pid, id); err != nil {
		return LatexDocument{}, err
	}
	doc, err := s.GetLatexDocument(id)
	if err == nil {
		s.mirrorLatexDocument(doc)
	}
	return doc, err
}

func (s *Store) DeleteLatexDocument(id string) error {
	doc, err := s.GetLatexDocument(id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = s.db.Exec(`DELETE FROM latex_documents WHERE id = ?`, id); err != nil {
		return err
	}
	if err == nil {
		s.removeLatexDocumentFiles(doc)
	}
	return nil
}
