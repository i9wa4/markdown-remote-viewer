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
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("plain text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notes.txt", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "plain text\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestNewRendersMarkdownAsHTMLPreview(t *testing.T) {
	dir := t.TempDir()
	source := "# Hello\n\nThis is **bold** and [linked](https://example.com).\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(source), 0o644); err != nil {
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
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want text/html", contentType)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Fatalf("content security policy = %q, want script execution disabled", csp)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<article class="markdown-body">`,
		`<h1>Hello</h1>`,
		`<strong>bold</strong>`,
		`<a href="https://example.com"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "# Hello") {
		t.Fatalf("body contains raw Markdown source:\n%s", body)
	}
}

func TestNewSanitizesMarkdownPreview(t *testing.T) {
	dir := t.TempDir()
	source := "# Unsafe\n\n<script>alert(1)</script>\n\n[bad](javascript:alert(1))\n"
	if err := os.WriteFile(filepath.Join(dir, "unsafe.md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unsafe.md", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, blocked := range []string{"<script", "alert(1)", "javascript:"} {
		if strings.Contains(body, blocked) {
			t.Fatalf("body contains unsafe %q:\n%s", blocked, body)
		}
	}
	if !strings.Contains(body, `<h1>Unsafe</h1>`) {
		t.Fatalf("body missing rendered safe content:\n%s", body)
	}
}

func TestNewKeepsMermaidFenceInert(t *testing.T) {
	dir := t.TempDir()
	source := "```mermaid\ngraph TD\n  A[<script>alert(1)</script>] --> B\n```\n"
	if err := os.WriteFile(filepath.Join(dir, "diagram.md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/diagram.md", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("body executed Mermaid fence as HTML:\n%s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("body missing escaped fence content:\n%s", body)
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

func TestNewDoesNotRenderMarkdownTraversalOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.md"), []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/../outside.md", nil)
	srv.Handler().ServeHTTP(rec, req)
	if location := rec.Header().Get("Location"); rec.Code >= 300 && rec.Code < 400 && location != "" {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, location, nil)
		srv.Handler().ServeHTTP(rec, req)
	}

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "Secret") {
		t.Fatalf("Markdown traversal rendered outside root: status=%d body=%q", rec.Code, rec.Body.String())
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
