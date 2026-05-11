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

func TestNewDoesNotServePathTraversalOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.txt"), []byte("TOP_SECRET_CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/../outside.txt", nil)
	srv.Handler().ServeHTTP(rec, req)
	if location := rec.Header().Get("Location"); rec.Code >= 300 && rec.Code < 400 && location != "" {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, location, nil)
		srv.Handler().ServeHTTP(rec, req)
	}

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "TOP_SECRET_CONTENT") {
		t.Fatalf("path traversal served outside root: status=%d body=%q", rec.Code, rec.Body.String())
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
