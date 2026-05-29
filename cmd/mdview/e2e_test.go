package main

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMDViewServesMarkdownDirectoryOverHTTP(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Hello\n\nThis is **bold**.\n\n<script>alert(1)</script>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("plain text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMDViewHelperProcess", "--", "--port", "0", root)
	cmd.Env = append(os.Environ(), "MDVIEW_E2E_HELPER=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer stopMDViewProcess(t, cmd, &stderr)

	baseURL := waitForStartupURL(t, stdout, &stderr)
	client := &http.Client{Timeout: 5 * time.Second}

	health := getHTTP(t, client, baseURL, "/healthz")
	if health.status != http.StatusNoContent {
		t.Fatalf("health status = %d, want %d", health.status, http.StatusNoContent)
	}

	page := getHTTP(t, client, baseURL, "/README.md")
	if page.status != http.StatusOK {
		t.Fatalf("Markdown status = %d, want %d", page.status, http.StatusOK)
	}
	if contentType := page.header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("Markdown content type = %q, want text/html", contentType)
	}
	if csp := page.header.Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("content security policy = %q, want same-origin script execution for Markdown preview", csp)
	}
	for _, want := range []string{
		`<article class="markdown-body">`,
		`<script defer src="/assets/markdown-viewer.js"></script>`,
		`<h1>Hello</h1>`,
		`<strong>bold</strong>`,
	} {
		if !strings.Contains(page.body, want) {
			t.Fatalf("Markdown body missing %q:\n%s", want, page.body)
		}
	}
	for _, blocked := range []string{"<script>alert", "alert(1)"} {
		if strings.Contains(page.body, blocked) {
			t.Fatalf("Markdown body contains unsafe %q:\n%s", blocked, page.body)
		}
	}

	asset := getHTTP(t, client, baseURL, "/assets/style.css")
	if asset.status != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", asset.status, http.StatusOK)
	}
	if !strings.Contains(asset.body, "color-scheme") {
		t.Fatalf("asset body missing stylesheet content:\n%s", asset.body)
	}

	viewerScript := getHTTP(t, client, baseURL, "/assets/markdown-viewer.js")
	if viewerScript.status != http.StatusOK {
		t.Fatalf("viewer script status = %d, want %d", viewerScript.status, http.StatusOK)
	}
	if !strings.Contains(viewerScript.body, "mermaid.min.js") {
		t.Fatalf("viewer script missing Mermaid loader:\n%s", viewerScript.body)
	}

	staticFile := getHTTP(t, client, baseURL, "/notes.txt")
	if staticFile.status != http.StatusOK {
		t.Fatalf("static file status = %d, want %d", staticFile.status, http.StatusOK)
	}
	if staticFile.body != "plain text\n" {
		t.Fatalf("static file body = %q", staticFile.body)
	}
}

func TestMDViewHelperProcess(t *testing.T) {
	if os.Getenv("MDVIEW_E2E_HELPER") != "1" {
		return
	}

	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"mdview"}, os.Args[i+1:]...)
			main()
			return
		}
	}
	os.Exit(2)
}

type httpResult struct {
	status int
	header http.Header
	body   string
}

func getHTTP(t *testing.T, client *http.Client, baseURL, path string) httpResult {
	t.Helper()

	resp, err := client.Get(resolveHTTPURL(t, baseURL, path))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return httpResult{
		status: resp.StatusCode,
		header: resp.Header,
		body:   string(body),
	}
}

func resolveHTTPURL(t *testing.T, baseURL, path string) string {
	t.Helper()

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	ref := &url.URL{Path: path}
	return parsedBase.ResolveReference(ref).String()
}

func waitForStartupURL(t *testing.T, stdout io.Reader, stderr *bytes.Buffer) string {
	t.Helper()

	type startupResult struct {
		url   string
		lines []string
		err   error
	}
	resultCh := make(chan startupResult, 1)
	go func() {
		var lines []string
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			lines = append(lines, line)
			if strings.HasPrefix(line, "URL: ") {
				resultCh <- startupResult{url: strings.TrimSpace(strings.TrimPrefix(line, "URL: ")), lines: lines}
				return
			}
		}
		resultCh <- startupResult{lines: lines, err: scanner.Err()}
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("read startup output: %v\nstdout=%q\nstderr=%q", result.err, strings.Join(result.lines, "\n"), stderr.String())
		}
		if result.url == "" {
			t.Fatalf("startup output did not include URL\nstdout=%q\nstderr=%q", strings.Join(result.lines, "\n"), stderr.String())
		}
		return result.url
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for startup URL\nstderr=%q", stderr.String())
		return ""
	}
}

func stopMDViewProcess(t *testing.T, cmd *exec.Cmd, stderr *bytes.Buffer) {
	t.Helper()

	if cmd.Process == nil {
		return
	}

	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		return
	case <-time.After(2 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("kill mdview process: %v\nstderr=%q", err, stderr.String())
		}
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("mdview process did not exit after kill\nstderr=%q", stderr.String())
	}
}
