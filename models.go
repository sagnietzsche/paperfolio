package main

import "time"

const (
	StatusUnread  = "unread"
	StatusReading = "reading"
	StatusRead    = "read"
)

type Paper struct {
	ID             string
	Title          string
	Authors        *string
	Abstract       *string
	Year           *int
	Venue          *string
	URL            *string
	PDFPath        *string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	HighlightCount int
	NoteCount      int
	IdeaCount      int
}

type PaperDraft struct {
	Title    string
	Authors  string
	Abstract string
	Year     *int
	Venue    string
	URL      string
}

// PaperPatch uses nil to leave a field alone. For text fields, a non-nil blank
// value clears the database column; this lets the form do what it displays.
type PaperPatch struct {
	Title    *string
	Authors  *string
	Abstract *string
	Year     *int
	Venue    *string
	URL      *string
	Status   *string
}

type Highlight struct {
	ID        string
	PaperID   string
	Content   string
	Comment   *string
	Emoji     *string
	Position  string
	Page      *int
	CreatedAt time.Time
}

type Note struct {
	ID        string
	PaperID   string
	Title     string
	Content   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Idea struct {
	ID        string
	PaperID   string
	Content   string
	CreatedAt time.Time
}

type LatexDocument struct {
	ID        string
	Title     string
	Content   string
	Template  string
	PaperID   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
