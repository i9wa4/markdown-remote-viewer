package cli

import (
	"bytes"
	"net"
	"net/http"
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
}

func TestRunStartsServerForExplicitPath(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer

	err := run([]string{dir}, &stdout, func(ln net.Listener, _ http.Handler) error {
		return ln.Close()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Clean(dir)) {
		t.Fatalf("startup output = %q, want root path", stdout.String())
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
