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
	fs := flag.NewFlagSet("mdview", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", "127.0.0.1", "address to bind")
	port := fs.Int("port", 0, "port to bind")
	tailscale := fs.Bool("tailscale", false, "bind to the Tailscale IPv4 address and print a Tailnet URL")
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
		fmt.Fprintf(stdout, "mdview %s (%s)\n", version.Version, version.Commit)
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

	listenAddr, displayHost, err := resolveServeAddress(serveAddressOptions{
		addr:         *addr,
		port:         *port,
		tailscale:    *tailscale,
		addrExplicit: flagWasSet(fs, "addr"),
	})
	if err != nil {
		return err
	}

	viewer, err := server.New(root)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	fmt.Fprintf(stdout, "Serving %s at http://%s/\n", viewer.Root(), displayAddress(displayHost, ln, *port))
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
	fmt.Fprint(w, `mdview serves a Markdown directory on a local HTTP server.

Usage:
  mdview [--addr ADDR | --tailscale] [--port PORT] [PATH]
  mdview --version
  mdview --help

Examples:
  mdview
  mdview docs
  mdview --port 8080 README-assets
  mdview --tailscale --port 8080 docs
`)
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	seen := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			seen = true
		}
	})
	return seen
}

func displayAddress(displayHost string, ln net.Listener, requestedPort int) string {
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil || port == "" {
		port = strconv.Itoa(requestedPort)
	}
	return net.JoinHostPort(displayHost, port)
}
