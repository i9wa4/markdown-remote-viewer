package cli

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStartsServerForDefaultDirectory(t *testing.T) {
	var stdout bytes.Buffer
	var called bool

	err := run([]string{"--port", "0"}, &stdout, func(ln net.Listener, handler http.Handler) error {
		called = true
		if ln.Addr().String() == "" {
			t.Fatal("listener address is empty")
		}
		if handler == nil {
			t.Fatal("handler is nil")
		}
		return ln.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("starter was not called")
	}
	if !strings.Contains(stdout.String(), "http://127.0.0.1:") {
		t.Fatalf("startup output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Access: loopback") {
		t.Fatalf("startup output = %q, want loopback access scope", stdout.String())
	}
}

func TestRunStartsServerForExplicitPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "docs")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer

	err := run([]string{dir}, &stdout, func(ln net.Listener, _ http.Handler) error {
		return ln.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if strings.Contains(got, filepath.Clean(dir)) {
		t.Fatalf("startup output = %q, want no absolute root path", got)
	}
	if !strings.Contains(got, "Serving docs\n") {
		t.Fatalf("startup output = %q, want directory basename", got)
	}
}

func TestDisplayRootAvoidsAbsolutePath(t *testing.T) {
	got := displayRoot("/home/example/private/docs")
	if got != "docs" {
		t.Fatalf("display root = %q, want basename", got)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer

	if err := run([]string{"--version"}, &stdout, nil); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.HasPrefix(got, "mdview ") {
		t.Fatalf("version output = %q", got)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer

	if err := run([]string{"--help"}, &stdout, nil); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("help output = %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "sanitized HTML previews") {
		t.Fatalf("help output = %q, want preview behavior", got)
	}
	if got := stdout.String(); !strings.Contains(got, "--open") {
		t.Fatalf("help output = %q, want open flag", got)
	}
}

func TestRunHelpDocumentsStableCommandSurface(t *testing.T) {
	var stdout bytes.Buffer

	if err := run([]string{"--help"}, &stdout, nil); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	for _, want := range []string{
		"mdview [--addr ADDR | --tailscale] [--port PORT] [--open] [--no-qr] [PATH]",
		"--addr ADDR",
		"--port PORT",
		"--open",
		"--tailscale",
		"--no-qr",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output = %q, want %q", got, want)
		}
	}
	if strings.Contains(got, "mdview share") {
		t.Fatalf("help output = %q, want flag-based command surface", got)
	}
}

func TestRunOpenBrowserUsesPrimaryURL(t *testing.T) {
	var stdout bytes.Buffer
	var openedURL string

	err := runWithBrowser([]string{"--open", "--port", "0"}, &stdout, ioDiscard{}, func(ln net.Listener, _ http.Handler) error {
		return ln.Close()
	}, func(url string) error {
		openedURL = url
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(openedURL, "http://127.0.0.1:") {
		t.Fatalf("opened URL = %q, want loopback URL", openedURL)
	}
	if !strings.HasSuffix(openedURL, "/") {
		t.Fatalf("opened URL = %q, want trailing slash", openedURL)
	}
	if !strings.Contains(stdout.String(), "URL: "+openedURL) {
		t.Fatalf("startup output = %q, want opened URL %q", stdout.String(), openedURL)
	}
}

func TestRunOpenBrowserFailureContinuesServing(t *testing.T) {
	var stderr bytes.Buffer
	var called bool

	err := runWithBrowser([]string{"--open", "--port", "0"}, ioDiscard{}, &stderr, func(ln net.Listener, _ http.Handler) error {
		called = true
		return ln.Close()
	}, func(string) error {
		return errors.New("no browser available")
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("starter was not called")
	}
	got := stderr.String()
	if !strings.Contains(got, "Could not open browser: no browser available") {
		t.Fatalf("stderr = %q, want browser failure", got)
	}
	if !strings.Contains(got, "Server is still running.") {
		t.Fatalf("stderr = %q, want continue-serving message", got)
	}
}

func TestRunRejectsTailscaleWithAddr(t *testing.T) {
	err := run([]string{"--tailscale", "--addr", "0.0.0.0"}, ioDiscard{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--tailscale cannot be used with --addr") {
		t.Fatalf("error = %q", err)
	}
}

func TestRunWildcardDisplayURLHidesUnspecifiedAddress(t *testing.T) {
	var stdout bytes.Buffer

	err := run([]string{"--addr", "0.0.0.0", "--port", "0"}, &stdout, func(ln net.Listener, handler http.Handler) error {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("health status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		return ln.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if strings.Contains(got, "0.0.0.0") || strings.Contains(got, "[::]") {
		t.Fatalf("startup output exposes unspecified host: %q", got)
	}
	if !strings.Contains(got, "http://") {
		t.Fatalf("startup output = %q, want URL", got)
	}
	if !strings.Contains(got, "Access: all interfaces") {
		t.Fatalf("startup output = %q, want all-interface access scope", got)
	}
}

func TestWriteStartupPrintsURLOnSeparateLine(t *testing.T) {
	var stdout bytes.Buffer
	root := "docs"

	writeStartup(&stdout, root, "Tailnet", []string{
		"100.89.157.2:8080",
	})

	got := stdout.String()
	if !strings.Contains(got, "Serving "+root+"\n") {
		t.Fatalf("startup output = %q, want root line", got)
	}
	if !strings.Contains(got, "\nURL: http://100.89.157.2:8080/\n") {
		t.Fatalf("startup output = %q, want primary URL on its own line", got)
	}
	if !strings.Contains(got, "\nAccess: Tailnet\n") {
		t.Fatalf("startup output = %q, want Tailnet access scope", got)
	}
	if strings.Contains(got, "Tailnet DNS") {
		t.Fatalf("startup output = %q, want no Tailnet DNS URL", got)
	}
}

func TestRunHelpDocumentsNoQRExample(t *testing.T) {
	var stdout bytes.Buffer

	if err := run([]string{"--help"}, &stdout, nil); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "--no-qr") {
		t.Fatalf("help output = %q, want no-qr example", got)
	}
}

func TestWriteStartupWithOptionsPrintsQRCode(t *testing.T) {
	var stdout bytes.Buffer

	writeStartupWithOptions(&stdout, "docs", "Tailnet", []string{
		"100.89.157.2:8080",
	}, startupOptions{showQR: true})

	got := stdout.String()
	if !strings.Contains(got, "\nQR:\n") {
		t.Fatalf("startup output = %q, want QR section", got)
	}
	if !strings.Contains(got, "##") {
		t.Fatalf("startup output = %q, want ASCII QR blocks", got)
	}
}

func TestWriteStartupWithOptionsCanOmitQRCode(t *testing.T) {
	var stdout bytes.Buffer

	writeStartupWithOptions(&stdout, "docs", "Tailnet", []string{
		"100.89.157.2:8080",
	}, startupOptions{showQR: false})

	got := stdout.String()
	if !strings.Contains(got, "\nURL: http://100.89.157.2:8080/\n") {
		t.Fatalf("startup output = %q, want URL", got)
	}
	if strings.Contains(got, "\nQR:\n") {
		t.Fatalf("startup output = %q, want no QR section", got)
	}
}

func TestRunRejectsTooManyPaths(t *testing.T) {
	err := run([]string{t.TempDir(), t.TempDir()}, ioDiscard{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRejectsMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	err := run([]string{path}, ioDiscard{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRejectsFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{path}, ioDiscard{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
