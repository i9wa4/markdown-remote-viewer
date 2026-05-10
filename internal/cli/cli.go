package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/i9wa4/markdown-remote-viewer/internal/server"
	"github.com/i9wa4/markdown-remote-viewer/internal/version"
)

type Starter func(net.Listener, http.Handler) error

func Run(args []string, _ io.Reader, stdout, _ io.Writer) error {
	return run(args, stdout, serve)
}

func run(args []string, stdout io.Writer, starter Starter) error {
	fs := flag.NewFlagSet("markserve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", "127.0.0.1", "address to bind")
	port := fs.Int("port", 0, "port to bind")
	showVersion := fs.Bool("version", false, "show version")
	showHelp := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(stdout)
			return nil
		}
		return err
	}

	if *showHelp {
		writeUsage(stdout)
		return nil
	}
	if *showVersion {
		fmt.Fprintf(stdout, "markserve %s (%s)\n", version.Version, version.Commit)
		return nil
	}

	paths := fs.Args()
	if len(paths) > 1 {
		return fmt.Errorf("expected at most one path, got %d", len(paths))
	}

	root := "."
	if len(paths) == 1 {
		root = paths[0]
	}

	viewer, err := server.New(root)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(*addr, strconv.Itoa(*port)))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	fmt.Fprintf(stdout, "Serving %s at http://%s/\n", viewer.Root(), ln.Addr().String())
	return starter(ln, viewer.Handler())
}

func serve(ln net.Listener, handler http.Handler) error {
	srv := &http.Server{Handler: handler}
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func writeUsage(w io.Writer) {
	fmt.Fprint(w, `markserve serves a Markdown directory on a local HTTP server.

Usage:
  markserve [--addr ADDR] [--port PORT] [PATH]
  markserve --version
  markserve --help

Examples:
  markserve
  markserve docs
  markserve --port 8080 README-assets
`)
}
