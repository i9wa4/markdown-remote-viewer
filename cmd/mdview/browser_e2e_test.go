//go:build browser_e2e

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestMDViewRendersMermaidDiagramInBrowser(t *testing.T) {
	root := t.TempDir()
	source := strings.Join([]string{
		"# Diagram",
		"",
		"```mermaid",
		"flowchart TD",
		"  A[Start] --> B[Finish]",
		"```",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(source), 0o644); err != nil {
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
	ctx, cancel := newBrowserContext(t)
	defer cancel()

	var svg string
	err = chromedp.Run(
		ctx,
		chromedp.Navigate(resolveHTTPURL(t, baseURL, "/README.md")),
		chromedp.WaitVisible(`.mdview-mermaid svg`, chromedp.ByQuery),
		chromedp.OuterHTML(`.mdview-mermaid svg`, &svg, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("render Mermaid diagram in browser: %v\nserver stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"<svg", "Start", "Finish"} {
		if !strings.Contains(svg, want) {
			t.Fatalf("rendered Mermaid SVG missing %q:\n%s", want, svg)
		}
	}
}

func newBrowserContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	browserPath := os.Getenv("MDVIEW_CHROME")
	if browserPath == "" {
		var err error
		browserPath, err = firstBrowserPath("chromium", "chromium-browser", "google-chrome", "google-chrome-stable")
		if err != nil {
			t.Fatalf("headless browser not found; run `nix develop .#ci --command go test -tags browser_e2e ./cmd/mdview`: %v", err)
		}
	}

	allocator, stopAllocator := chromedp.NewExecAllocator(
		context.Background(),
		append(
			chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(browserPath),
			chromedp.NoSandbox,
		)...,
	)
	ctx, stopBrowser := chromedp.NewContext(allocator)
	ctx, stopTimeout := context.WithTimeout(ctx, 15*time.Second)

	return ctx, func() {
		stopTimeout()
		stopBrowser()
		stopAllocator()
	}
}

func firstBrowserPath(names ...string) (string, error) {
	var lastErr error
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}
	return "", lastErr
}
