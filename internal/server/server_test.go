package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewServesFilesUnderRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/README.md", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "# Hello\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestNewServesEmbeddedAssets(t *testing.T) {
	srv, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "color-scheme") {
		t.Fatalf("embedded stylesheet was not served: %q", rec.Body.String())
	}
}

func TestNewRejectsNonDirectoryRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := New(path); err == nil {
		t.Fatal("expected error for file root")
	}
}
