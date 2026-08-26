# Paperfolio

Paperfolio is a native desktop app for reading research papers and keeping what
you learn from them. Import PDFs, track reading status, capture highlights,
write markdown notes, and save quick ideas. Everything stays local in a folder
that you can open, copy, and back up yourself.

## Features

- Paper metadata: title, authors, abstract, year, venue, URL, and status.
- PDF import and replacement, stored under the library's `uploads/` directory.
- Highlights with quotes, optional notes, and page numbers.
- Markdown notes mirrored to plain `.md` files.
- One-line ideas per paper.
- **LaTeX Writing Studio**: compose academic papers, theses, slides, and letters with templates (article, IEEE, report, beamer, letter, minimal), snippet insertion, live KaTeX math preview, `.tex` export, and `pdflatex`/`xelatex`/`lualatex` compile-to-PDF when a TeX distribution is installed. Documents are mirrored to `writing/*.tex` and can be linked to papers.
- Cascade deletion: deleting a paper removes its annotations and imported PDF.
- No account, sync service, or remote PDF service.
- In-app PDF rendering through a local embedded webview and bundled PDF.js.
- In-app LaTeX preview through the same embedded webview (KaTeX via CDN, offline fallback shows raw source).

Paperfolio renders imported PDFs inside the application. The embedded webview
uses the platform's system web runtime; no PDF is sent to a remote service.

## Requirements

- Go 1.22 or newer.
- A Fyne-supported desktop platform and its native build prerequisites. On macOS,
  install Xcode Command Line Tools with `xcode-select --install`.
- CGO enabled (required by SQLite, Fyne's desktop backends, and the embedded webview).
- A supported system web runtime for the embedded webview. On macOS, this is WebKit;
  Windows uses WebView2; Linux requires the webview's GTK/WebKit dependencies.
- Optional: a LaTeX distribution (TeX Live, MiKTeX, or MacTeX) for **Compile PDF** in the Writing Studio (`pdflatex`, `xelatex`, or `lualatex` on PATH). Without it, you can still author and export `.tex` files and compile externally.

## Running

```bash
go mod download
go run .
```

Build a native executable into `./bin` with Task:

```bash
task build
```

Run the test suite with:

```bash
task test
```

You can also run the application with `task run` and remove build artifacts with
`task clean`.

## Your data

The app creates this visible folder on first launch:

```
~/Documents/Paperfolio_Data/
├── paperfolio.db      # SQLite database, WAL mode
├── uploads/           # imported PDFs
├── notes/             # markdown mirror, one folder per paper
└── writing/           # LaTeX mirror, one .tex per document (writing/<slug>-<id>.tex)
```

That folder is the whole library. Quit Paperfolio before copying it so SQLite
can checkpoint its write-ahead log. Copy it to back up or move the library to
another machine. Delete it to start over.

Note files use readable names such as
`notes/attention-is-all-you-need-a1b2c3d4/reading-notes-e5f6a7b8.md` and contain
small YAML-style front matter followed by the markdown body.

## Project layout

```
├── main.go        # application startup and data directory
├── app.go         # Fyne windows, forms, tabs, and native actions
├── app_latex.go   # LaTeX Writing Studio UI and preview server
├── latex.go       # LaTeX templates, snippets, and file mirroring
├── models.go      # domain models and patch types
├── store.go       # SQLite schema and persistence
├── markdown.go    # markdown mirror and filesystem-safe slugs
├── store_test.go   # storage, validation, import, and cleanup tests
└── website/       # static project landing page
```

## Architecture notes

The desktop application is Go + Fyne. SQLite is embedded through
`github.com/mattn/go-sqlite3`; no database server is needed. Fyne provides the
native widgets and window lifecycle. The embedded webview is supplied by
`apptrix.org/components/widget/webview`, and PDF files are rendered locally by
the bundled PDF.js 4.10.38 viewer through the platform web runtime. Imported files are copied
into the application-owned library, so source files can be moved or deleted
afterward.

The former Tauri/Rust/React/TypeScript implementation has been removed.The application does not use a remote browser runtime or PDF service. PDF.js
runtime and worker assets are checked into `assets/pdfjs` and embedded into the
application binary. The webview
is only used for the local PDF.js reader and the selected PDF is loaded from the
managed Paperfolio uploads directory.

## License

MIT.
