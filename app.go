package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"apptrix.org/components/widget/webview"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type paperfolioApp struct {
	app         fyne.App
	win         fyne.Window
	store       *Store
	pdfView     webview.WebView
	pdfServer   *pdfViewerServer
	latexView   webview.WebView
	latexServer *latexPreviewServer
}

func newPaperfolioApp(a fyne.App, store *Store) *paperfolioApp {
	p := &paperfolioApp{app: a, store: store}
	p.win = a.NewWindow("Paperfolio")
	p.win.Resize(fyne.NewSize(1280, 860))
	p.win.SetOnClosed(func() {
		p.closePDFViewer()
		p.closeLatexPreview()
		_ = store.Close()
	})
	return p
}

func (p *paperfolioApp) showError(err error) {
	if err != nil {
		dialog.ShowError(err, p.win)
	}
}

func (p *paperfolioApp) showWelcome() {
	papers, err := p.store.ListPapers()
	if err != nil {
		p.showError(err)
		return
	}
	docs, _ := p.store.ListLatexDocuments()
	stats := fmt.Sprintf("%d papers  ·  %d LaTeX documents", len(papers), len(docs))
	if len(papers) == 0 && len(docs) == 0 {
		stats = "Your library is empty"
	}
	content := container.NewVBox(
		layout.NewSpacer(),
		widget.NewLabelWithStyle("▤", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Paperfolio", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Read, highlight and annotate research papers.\nEvery paper gets its own space for its PDF, highlights, notes and ideas.\nWrite academic papers in the integrated LaTeX editor.", fyne.TextAlignCenter, fyne.TextStyle{}),
		container.NewHBox(
			layout.NewSpacer(),
			widget.NewButtonWithIcon(func() string {
				if len(papers) == 0 {
					return "Add your first paper"
				}
				return "Open library"
			}(), theme.NavigateNextIcon(), func() { p.showLibrary() }),
			widget.NewButtonWithIcon("Writing Studio", theme.DocumentCreateIcon(), func() { p.showWritingStudio() }),
			layout.NewSpacer(),
		),
		widget.NewLabelWithStyle(stats, fyne.TextAlignCenter, fyne.TextStyle{}),
		layout.NewSpacer(),
		widget.NewLabelWithStyle("Stored in Documents / Paperfolio_Data  (papers.db + uploads/ + notes/ + writing/)", fyne.TextAlignCenter, fyne.TextStyle{}),
	)
	p.win.SetContent(container.NewMax(content))
}

func (p *paperfolioApp) showLibrary() {
	papers, err := p.store.ListPapers()
	if err != nil {
		p.showError(err)
		return
	}
	add := widget.NewButtonWithIcon("Add paper", theme.ContentAddIcon(), func() { p.showPaperForm(nil) })
	header := container.NewBorder(nil, nil,
		container.NewHBox(widget.NewButtonWithIcon("Home", theme.HomeIcon(), p.showWelcome), widget.NewButtonWithIcon("Writing Studio", theme.DocumentCreateIcon(), func() { p.showWritingStudio() })),
		add, widget.NewLabelWithStyle("Paperfolio  ·  Library", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	var cards []fyne.CanvasObject
	for _, paper := range papers {
		cards = append(cards, p.paperCard(paper))
	}
	var body fyne.CanvasObject
	if len(cards) == 0 {
		body = container.NewCenter(container.NewVBox(widget.NewLabelWithStyle("Your library is empty.", fyne.TextAlignCenter, fyne.TextStyle{}), widget.NewButton("Add your first paper", func() { p.showPaperForm(nil) })))
	} else {
		body = container.NewVScroll(container.NewVBox(cards...))
	}
	p.win.SetContent(container.NewBorder(container.NewPadded(header), nil, nil, nil, container.NewPadded(body)))
}

func (p *paperfolioApp) paperCard(paper *Paper) fyne.CanvasObject {
	status := widget.NewLabel(strings.Title(paper.Status))
	meta := []string{}
	if paper.Authors != nil {
		meta = append(meta, *paper.Authors)
	}
	if paper.Venue != nil {
		meta = append(meta, *paper.Venue)
	}
	if paper.Year != nil {
		meta = append(meta, strconv.Itoa(*paper.Year))
	}
	details := paper.Title
	if len(meta) > 0 {
		details += "\n" + strings.Join(meta, "  ·  ")
	}
	counts := widget.NewLabel(fmt.Sprintf("%d highlights  ·  %d notes  ·  %d ideas", paper.HighlightCount, paper.NoteCount, paper.IdeaCount))
	open := widget.NewButton("Open", func() { p.showPaper(paper.ID) })
	return container.NewBorder(nil, counts, status, open, widget.NewLabel(details))
}

func (p *paperfolioApp) showPaperForm(existing *Paper) {
	title := widget.NewEntry()
	authors := widget.NewEntry()
	year := widget.NewEntry()
	venue := widget.NewEntry()
	url := widget.NewEntry()
	abstract := widget.NewMultiLineEntry()
	heading, confirmLabel := "Add a paper", "Add paper"
	if existing != nil {
		heading, confirmLabel = "Edit details", "Save changes"
		title.SetText(existing.Title)
		setEntry(authors, existing.Authors)
		setEntry(venue, existing.Venue)
		setEntry(url, existing.URL)
		setEntry(abstract, existing.Abstract)
		if existing.Year != nil {
			year.SetText(strconv.Itoa(*existing.Year))
		}
	}
	var pdfData []byte
	pdfName := widget.NewLabel("No PDF chosen")
	choose := widget.NewButton("Choose PDF…", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				p.showError(err)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			data, readErr := io.ReadAll(reader)
			if readErr != nil {
				p.showError(readErr)
				return
			}
			if len(data) < 5 || string(data[:5]) != "%PDF-" {
				p.showError(fmt.Errorf("only PDF files are allowed"))
				return
			}
			pdfData = data
			name := "Selected PDF"
			if reader.URI() != nil {
				name = filepath.Base(reader.URI().Path())
			}
			pdfName.SetText(name)
		}, p.win)
	})
	formItems := []*widget.FormItem{
		widget.NewFormItem("Title *", title), widget.NewFormItem("Authors", authors),
		widget.NewFormItem("Year", year), widget.NewFormItem("Venue", venue), widget.NewFormItem("URL", url),
		widget.NewFormItem("Abstract", abstract),
	}
	if existing == nil {
		formItems = append(formItems, widget.NewFormItem("PDF", container.NewBorder(nil, nil, nil, pdfName, choose)))
	}
	form := widget.NewForm(formItems...)
	d := dialog.NewCustomConfirm(heading, confirmLabel, "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		yearValue, err := parseYear(year.Text)
		if err != nil {
			p.showError(err)
			return
		}
		if strings.TrimSpace(title.Text) == "" {
			p.showError(fmt.Errorf("title is required"))
			return
		}
		if existing == nil {
			var pdf io.Reader
			if len(pdfData) > 0 {
				pdf = bytes.NewReader(pdfData)
			}
			_, err = p.store.CreatePaper(PaperDraft{Title: title.Text, Authors: authors.Text, Abstract: abstract.Text, Year: yearValue, Venue: venue.Text, URL: url.Text}, pdf)
		} else {
			_, err = p.store.UpdatePaper(existing.ID, PaperPatch{Title: &title.Text, Authors: &authors.Text, Abstract: &abstract.Text, Year: yearValue, Venue: &venue.Text, URL: &url.Text})
		}
		if err != nil {
			p.showError(err)
			return
		}
		p.showLibrary()
	}, p.win)
	d.Resize(fyne.NewSize(600, 650))
	d.Show()
}

func setEntry(entry *widget.Entry, value *string) {
	if value != nil {
		entry.SetText(*value)
	}
}
func parseYear(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	year, err := strconv.Atoi(value)
	if err != nil || year < 1900 || year > 2100 {
		return nil, fmt.Errorf("year must be between 1900 and 2100")
	}
	return &year, nil
}

func (p *paperfolioApp) showPaper(id string) {
	paper, err := p.store.GetPaper(id)
	if err != nil {
		p.showError(err)
		return
	}
	highlights, err := p.store.ListHighlights(id)
	if err != nil {
		p.showError(err)
		return
	}
	notes, err := p.store.ListNotes(id)
	if err != nil {
		p.showError(err)
		return
	}
	ideas, err := p.store.ListIdeas(id)
	if err != nil {
		p.showError(err)
		return
	}

	back := widget.NewButtonWithIcon("Library", theme.NavigateBackIcon(), p.showLibrary)
	status := widget.NewSelect([]string{"Unread", "Reading", "Read"}, func(value string) {
		statusValue := strings.ToLower(value)
		_, updateErr := p.store.UpdatePaper(id, PaperPatch{Status: &statusValue})
		if updateErr != nil {
			p.showError(updateErr)
		}
	})
	status.SetSelected(strings.Title(paper.Status))
	headerLeft := container.NewBorder(nil, nil, back, nil, widget.NewLabelWithStyle(paper.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	controls := container.NewHBox(status, widget.NewButton("Edit details", func() { p.showPaperForm(paper) }), widget.NewButton("Replace PDF", func() { p.chooseReplacement(paper.ID) }), widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() { p.deletePaper(paper) }))
	header := container.NewVBox(headerLeft, paperMeta(paper), controls)
	tabs := container.NewAppTabs(container.NewTabItem("Reader", p.readerTab(paper, highlights)), container.NewTabItem("Notes", p.notesTab(paper, notes)), container.NewTabItem("Ideas", p.ideasTab(paper, ideas)), container.NewTabItem("Writing", p.paperLatexTab(paper)))
	p.win.SetContent(container.NewBorder(container.NewPadded(header), nil, nil, nil, tabs))
}

func paperMeta(paper *Paper) fyne.CanvasObject {
	parts := []string{}
	if paper.Authors != nil {
		parts = append(parts, *paper.Authors)
	}
	if paper.Venue != nil {
		parts = append(parts, *paper.Venue)
	}
	if paper.Year != nil {
		parts = append(parts, strconv.Itoa(*paper.Year))
	}
	if len(parts) == 0 {
		return widget.NewLabel("No bibliographic details yet")
	}
	return widget.NewLabel(strings.Join(parts, "  ·  "))
}

func (p *paperfolioApp) readerTab(paper *Paper, highlights []Highlight) fyne.CanvasObject {
	var top fyne.CanvasObject
	if paper.PDFPath == nil {
		top = container.NewVBox(widget.NewLabelWithStyle("No PDF added yet.", fyne.TextAlignCenter, fyne.TextStyle{}), widget.NewButton("Choose a PDF…", func() { p.chooseReplacement(paper.ID) }))
	} else {
		top = p.embeddedPDFViewer(paper)
	}
	add := widget.NewButtonWithIcon("Add highlight", theme.ContentAddIcon(), func() { p.showHighlightForm(paper.ID) })
	items := []fyne.CanvasObject{container.NewBorder(nil, nil, nil, add, widget.NewLabelWithStyle("Highlights", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))}
	for _, h := range highlights {
		items = append(items, p.highlightCard(paper.ID, h))
	}
	if len(highlights) == 0 {
		items = append(items, widget.NewLabel("No highlights yet. Add a quote and page number while reading."))
	}
	return container.NewBorder(container.NewPadded(top), nil, nil, nil, container.NewVScroll(container.NewVBox(items...)))
}

func (p *paperfolioApp) highlightCard(paperID string, h Highlight) fyne.CanvasObject {
	page := "page unknown"
	if h.Page != nil {
		page = fmt.Sprintf("page %d", *h.Page)
	}
	quote := widget.NewLabel(fmt.Sprintf("“%s”\n%s", h.Content, page))
	quote.Wrapping = fyne.TextWrapWord
	edit := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() { p.showHighlightEditor(paperID, h) })
	remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if err := p.store.DeleteHighlight(h.ID); err != nil {
			p.showError(err)
		}
		p.showPaper(paperID)
	})
	return container.NewBorder(nil, nil, nil, container.NewHBox(edit, remove), quote)
}

func (p *paperfolioApp) showHighlightEditor(paperID string, highlight Highlight) {
	comment := widget.NewMultiLineEntry()
	if highlight.Comment != nil {
		comment.SetText(*highlight.Comment)
	}
	form := widget.NewForm(widget.NewFormItem("Note", comment))
	d := dialog.NewCustomConfirm("Edit highlight note", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		if _, err := p.store.UpdateHighlight(highlight.ID, cleanText(comment.Text), highlight.Emoji); err != nil {
			p.showError(err)
			return
		}
		p.showPaper(paperID)
	}, p.win)
	d.Resize(fyne.NewSize(500, 300))
	d.Show()
}

func (p *paperfolioApp) showHighlightForm(paperID string) {
	quote, comment, page := widget.NewMultiLineEntry(), widget.NewMultiLineEntry(), widget.NewEntry()
	page.SetText("1")
	form := widget.NewForm(widget.NewFormItem("Quote", quote), widget.NewFormItem("Note", comment), widget.NewFormItem("Page", page))
	d := dialog.NewCustomConfirm("Add highlight", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		pageNumber, err := strconv.Atoi(strings.TrimSpace(page.Text))
		if err != nil || pageNumber < 1 {
			p.showError(fmt.Errorf("page must be a positive number"))
			return
		}
		position, _ := json.Marshal(map[string]int{"pageNumber": pageNumber})
		_, err = p.store.CreateHighlight(Highlight{PaperID: paperID, Content: quote.Text, Comment: cleanText(comment.Text), Position: string(position), Page: &pageNumber})
		if err != nil {
			p.showError(err)
			return
		}
		p.showPaper(paperID)
	}, p.win)
	d.Resize(fyne.NewSize(560, 460))
	d.Show()
}

func (p *paperfolioApp) notesTab(paper *Paper, notes []Note) fyne.CanvasObject {
	title, content := widget.NewEntry(), widget.NewMultiLineEntry()
	add := widget.NewButtonWithIcon("Add note", theme.ContentAddIcon(), func() {
		note, err := p.store.CreateNote(paper.ID, title.Text, content.Text)
		if err != nil {
			p.showError(err)
			return
		}
		_ = note
		p.showPaper(paper.ID)
	})
	composer := container.NewVBox(widget.NewLabelWithStyle("New markdown note", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), title, content, add)
	items := []fyne.CanvasObject{composer}
	for _, note := range notes {
		items = append(items, p.noteCard(*paper, note))
	}
	if len(notes) == 0 {
		items = append(items, widget.NewLabel("No notes yet."))
	}
	return container.NewPadded(container.NewVScroll(container.NewVBox(items...)))
}

func (p *paperfolioApp) noteCard(paper Paper, note Note) fyne.CanvasObject {
	body := ""
	if note.Content != nil {
		body = *note.Content
	}
	content := widget.NewRichTextFromMarkdown(body)
	edit := widget.NewButton("Edit", func() { p.showNoteEditor(paper.ID, note) })
	remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if err := p.store.DeleteNote(note.ID); err != nil {
			p.showError(err)
			return
		}
		p.showPaper(paper.ID)
	})
	return container.NewVBox(container.NewBorder(nil, nil, widget.NewLabelWithStyle(note.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), container.NewHBox(edit, remove), nil), content, widget.NewSeparator())
}

func (p *paperfolioApp) showNoteEditor(paperID string, note Note) {
	title, content := widget.NewEntry(), widget.NewMultiLineEntry()
	title.SetText(note.Title)
	if note.Content != nil {
		content.SetText(*note.Content)
	}
	form := widget.NewForm(widget.NewFormItem("Title", title), widget.NewFormItem("Markdown", content))
	d := dialog.NewCustomConfirm("Edit note", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		if _, err := p.store.UpdateNote(note.ID, title.Text, content.Text); err != nil {
			p.showError(err)
			return
		}
		p.showPaper(paperID)
	}, p.win)
	d.Resize(fyne.NewSize(650, 560))
	d.Show()
}

func (p *paperfolioApp) ideasTab(paper *Paper, ideas []Idea) fyne.CanvasObject {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Capture an idea…")
	add := widget.NewButtonWithIcon("Add idea", theme.ContentAddIcon(), func() {
		if _, err := p.store.CreateIdea(paper.ID, entry.Text); err != nil {
			p.showError(err)
			return
		}
		p.showPaper(paper.ID)
	})
	items := []fyne.CanvasObject{container.NewBorder(nil, nil, nil, add, entry)}
	for _, idea := range ideas {
		remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func(id string) func() {
			return func() {
				if err := p.store.DeleteIdea(id); err != nil {
					p.showError(err)
					return
				}
				p.showPaper(paper.ID)
			}
		}(idea.ID))
		items = append(items, container.NewBorder(nil, nil, nil, remove, widget.NewLabel(idea.Content)))
	}
	if len(ideas) == 0 {
		items = append(items, widget.NewLabel("No ideas captured yet."))
	}
	return container.NewPadded(container.NewVScroll(container.NewVBox(items...)))
}

func (p *paperfolioApp) chooseReplacement(paperID string) {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			p.showError(err)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			p.showError(readErr)
			return
		}
		if len(data) < 5 || string(data[:5]) != "%PDF-" {
			p.showError(fmt.Errorf("only PDF files are allowed"))
			return
		}
		if _, err = p.store.SetPaperPDF(paperID, bytes.NewReader(data)); err != nil {
			p.showError(err)
			return
		}
		p.showPaper(paperID)
	}, p.win)
}

func (p *paperfolioApp) deletePaper(paper *Paper) {
	dialog.ShowConfirm("Delete this paper and all of its highlights, notes and ideas?", "Delete paper", func(ok bool) {
		if !ok {
			return
		}
		if err := p.store.DeletePaper(paper.ID); err != nil {
			p.showError(err)
			return
		}
		p.showLibrary()
	}, p.win)
}

func (p *paperfolioApp) embeddedPDFViewer(paper *Paper) fyne.CanvasObject {
	path, err := p.store.PDFAbsolutePath(paper)
	if err != nil {
		return container.NewVBox(widget.NewLabelWithStyle("Unable to load PDF.", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), widget.NewLabel(err.Error()))
	}
	p.closePDFViewer()
	server, err := startPDFViewerServer(p.store, paper)
	if err != nil {
		return container.NewVBox(widget.NewLabelWithStyle("Unable to start embedded PDF viewer.", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), widget.NewLabel(err.Error()))
	}
	p.pdfServer = server
	viewer, err := webview.New(p.win)
	if err != nil {
		return container.NewVBox(widget.NewLabelWithStyle("Embedded PDF viewer unavailable.", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}), widget.NewLabel(err.Error()))
	}
	p.pdfView = viewer
	viewerURL, err := server.documentURL(filepath.Base(path))
	if err != nil {
		viewer.Close()
		p.pdfView = nil
		server.Close()
		p.pdfServer = nil
		return widget.NewLabel(err.Error())
	}
	viewer.Load(viewerURL)
	return viewer
}

func (p *paperfolioApp) closePDFViewer() {
	if p.pdfView != nil {
		p.pdfView.Close()
		p.pdfView = nil
	}
	if p.pdfServer != nil {
		_ = p.pdfServer.Close()
		p.pdfServer = nil
	}
}
