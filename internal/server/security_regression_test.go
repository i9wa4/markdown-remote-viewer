package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityRegressionDoesNotServeEncodedTraversalOutsideRoot(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/%2e%2e/outside.txt", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "TOP_SECRET_CONTENT") {
		t.Fatalf("encoded traversal served outside root: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSecurityRegressionDoesNotServeSymlinkEscapeOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(outside, []byte("TOP_SECRET_CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/link.txt", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "TOP_SECRET_CONTENT") {
		t.Fatalf("symlink escape served outside root: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSecurityRegressionDoesNotServeRootIndexSymlinkEscapeOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.html")
	if err := os.WriteFile(outside, []byte("TOP_SECRET_CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "index.html")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "TOP_SECRET_CONTENT") {
		t.Fatalf("root index symlink escape served outside root: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSecurityRegressionDoesNotServeNestedIndexSymlinkEscapeOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.html")
	if err := os.WriteFile(outside, []byte("TOP_SECRET_CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "sub", "index.html")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sub/", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "TOP_SECRET_CONTENT") {
		t.Fatalf("nested index symlink escape served outside root: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSecurityRegressionDoesNotRenderMarkdownSymlinkEscapeOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "public")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.md")
	if err := os.WriteFile(outside, []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/link.md", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "Secret") {
		t.Fatalf("Markdown symlink escape rendered outside root: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSecurityRegressionServesSymlinkContainedWithinRoot(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(docs, "notes.txt")
	if err := os.WriteFile(target, []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/link.txt", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "inside\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestSecurityRegressionServesRootIndexInsideRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Home</h1>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "<h1>Home</h1>\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestSecurityRegressionServesDirectoryListingWithoutIndex(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "notes.txt"), []byte("plain text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "notes.txt") {
		t.Fatalf("directory listing missing child: %q", rec.Body.String())
	}
}

func TestSecurityRegressionServesStaticThroughSymlinkedDirectoryInsideRoot(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "inside")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inside, "notes.txt"), []byte("plain text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, filepath.Join(dir, "linked-dir")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/linked-dir/notes.txt", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "plain text\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestSecurityRegressionRendersMarkdownThroughSymlinkedDirectoryInsideRoot(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "inside")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inside, "README.md"), []byte("# Linked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, filepath.Join(dir, "linked-dir")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/linked-dir/README.md", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `<h1>Linked</h1>`) {
		t.Fatalf("body missing rendered Markdown:\n%s", rec.Body.String())
	}
}
