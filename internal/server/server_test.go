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
	assertSecurityHeaders(t, rec)
}

func TestNewServesReadOnlyDirectoryIndex(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"README.md": "# Hello\n",
		"notes.txt": "plain text\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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
	body := rec.Body.String()
	for _, want := range []string{
		`href="README.md"`,
		`href="notes.txt"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("directory index missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{
		"<form",
		"method=",
		"upload",
		"delete",
	} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("directory index exposes write affordance %q:\n%s", forbidden, body)
		}
	}
	assertSecurityHeaders(t, rec)
}

func TestNewRejectsWriteMethods(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/README.md", strings.NewReader("mutate"))
			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
			assertSecurityHeaders(t, rec)
		})
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
	assertSecurityHeaders(t, rec)
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

func TestNewRendersGitHubFlavoredMarkdownBasics(t *testing.T) {
	dir := t.TempDir()
	source := strings.Join([]string{
		"## Basics",
		"",
		"- plain item",
		"- [x] completed task",
		"",
		"| Name | Value |",
		"| ---- | ----- |",
		"| mdview | ready |",
		"",
		"```go",
		"return nil",
		"```",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "basics.md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/basics.md", nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<h2>Basics</h2>`,
		`<li>plain item</li>`,
		`<input checked="" disabled="" type="checkbox">`,
		`<table>`,
		`<th>Name</th>`,
		`<td>mdview</td>`,
		`<pre><code class="language-go">return nil`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "| Name | Value |") {
		t.Fatalf("body contains raw table Markdown:\n%s", body)
	}
}

func TestNewSanitizesMarkdownPreview(t *testing.T) {
	dir := t.TempDir()
	source := "# Unsafe\n\n<script>alert(1)</script>\n\n[bad](javascript:alert(1))\n\n<img src=x onerror=alert(2)>\n"
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
	for _, blocked := range []string{"<script", "alert(1)", "alert(2)", "javascript:", "onerror"} {
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

func TestNewDoesNotServeEncodedTraversalOutsideRoot(t *testing.T) {
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

func TestNewDoesNotServeSymlinkEscapeOutsideRoot(t *testing.T) {
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

func TestNewDoesNotServeRootIndexSymlinkEscapeOutsideRoot(t *testing.T) {
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

func TestNewDoesNotServeNestedIndexSymlinkEscapeOutsideRoot(t *testing.T) {
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

func TestNewDoesNotRenderMarkdownSymlinkEscapeOutsideRoot(t *testing.T) {
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

func TestNewServesSymlinkContainedWithinRoot(t *testing.T) {
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

func TestNewServesRootIndexInsideRoot(t *testing.T) {
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

func TestNewServesDirectoryListingWithoutIndex(t *testing.T) {
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

func TestNewServesStaticThroughSymlinkedDirectoryInsideRoot(t *testing.T) {
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

func TestNewRendersMarkdownThroughSymlinkedDirectoryInsideRoot(t *testing.T) {
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
	assertSecurityHeaders(t, rec)
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

func assertSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'none'") {
		t.Fatalf("content security policy = %q, want script execution disabled", csp)
	}
	if nosniff := rec.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Fatalf("x-content-type-options = %q, want nosniff", nosniff)
	}
}
