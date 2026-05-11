package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
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

	serveAddr, err := resolveServeAddress(serveAddressOptions{
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

	ln, err := net.Listen("tcp", serveAddr.listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	writeStartup(stdout, displayRoot(root), displayAddresses(serveAddr.displayHosts, ln, *port))
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

func displayAddresses(displayHosts []string, ln net.Listener, requestedPort int) []string {
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil || port == "" {
		port = strconv.Itoa(requestedPort)
	}
	addrs := make([]string, 0, len(displayHosts))
	for _, host := range displayHosts {
		addrs = append(addrs, net.JoinHostPort(host, port))
	}
	return addrs
}

func displayRoot(root string) string {
	if root == "" {
		return "."
	}
	clean := filepath.Clean(root)
	if !filepath.IsAbs(clean) {
		return clean
	}
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) {
		return "selected directory"
	}
	return base
}

func writeStartup(stdout io.Writer, root string, addresses []string) {
	fmt.Fprintf(stdout, "Serving %s\n", root)
	for _, addr := range addresses {
		fmt.Fprintf(stdout, "URL: http://%s/\n", addr)
	}
}
