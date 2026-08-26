package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPDFAbsolutePath(t *testing.T) {
	root := t.TempDir()
	store := &Store{UploadsDir: filepath.Join(root, "uploads")}
	if err := os.MkdirAll(store.UploadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.UploadsDir, "paper.pdf"), []byte("%PDF-1.7"), 0o644); err != nil {
		t.Fatal(err)
	}
	name := "paper.pdf"
	path, err := store.PDFAbsolutePath(&Paper{PDFPath: &name})
	if err != nil {
		t.Fatalf("expected valid PDF path: %v", err)
	}
	if path != filepath.Join(store.UploadsDir, name) {
		t.Fatalf("unexpected path %q", path)
	}
}

func TestPDFAbsolutePathRejectsMissingAndDirectories(t *testing.T) {
	root := t.TempDir()
	store := &Store{UploadsDir: filepath.Join(root, "uploads")}
	if err := os.MkdirAll(filepath.Join(store.UploadsDir, "folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing.pdf", "folder"} {
		name := name
		if _, err := store.PDFAbsolutePath(&Paper{PDFPath: &name}); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestBundledPDFJSAssets(t *testing.T) {
	if len(bundledPDFJS) == 0 || len(bundledPDFJSWorker) == 0 {
		t.Fatal("bundled PDF.js assets must be non-empty")
	}
}

func TestFileURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paper file.pdf")
	u, err := fileURL(path)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "file" || u.Path == "" {
		t.Fatalf("unexpected file URL: %v", u)
	}
}
