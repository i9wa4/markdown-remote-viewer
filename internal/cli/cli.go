package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/i9wa4/markdown-remote-viewer/internal/server"
	"github.com/i9wa4/markdown-remote-viewer/internal/version"
	qrcode "github.com/skip2/go-qrcode"
)

type Starter func(net.Listener, http.Handler) error

func Run(args []string, _ io.Reader, stdout, stderr io.Writer) error {
	return runWithBrowser(args, stdout, stderr, serve, defaultBrowserOpener)
}

func run(args []string, stdout io.Writer, starter Starter) error {
	return runWithBrowser(args, stdout, io.Discard, starter, defaultBrowserOpener)
}

func runWithBrowser(args []string, stdout, stderr io.Writer, starter Starter, openBrowser BrowserOpener) error {
	fs := flag.NewFlagSet("mdview", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", "127.0.0.1", "address to bind")
	port := fs.Int("port", 0, "port to bind")
	open := fs.Bool("open", false, "open the primary URL in the local browser")
	tailscale := fs.Bool("tailscale", false, "bind to the Tailscale IPv4 address and print a Tailnet URL")
	noQR := fs.Bool("no-qr", false, "disable terminal QR output in Tailnet mode")
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

	if *open {
		starter = starterWithBrowserOpen(starter, stderr, openBrowser, serveAddr.displayHosts, *port)
	}

	viewer, err := server.New(root)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", serveAddr.listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	addresses := displayAddresses(serveAddr.displayHosts, ln, *port)
	writeStartupWithOptions(stdout, displayRoot(root), serveAddr.access, addresses, startupOptions{
		showQR: *tailscale && !*noQR && writerIsTerminal(stdout),
	})
	return starter(ln, viewer.Handler())
}

func serve(ln net.Listener, handler http.Handler) error {
	srv := &http.Server{Handler: handler}
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func starterWithBrowserOpen(starter Starter, stderr io.Writer, openBrowser BrowserOpener, displayHosts []string, requestedPort int) Starter {
	return func(ln net.Listener, handler http.Handler) error {
		addresses := displayAddresses(displayHosts, ln, requestedPort)
		if err := openPrimaryURL(openBrowser, addresses); err != nil {
			fmt.Fprintf(stderr, "Could not open browser: %v\n", err)
			if len(addresses) > 0 {
				fmt.Fprintf(stderr, "Server is still running. Open http://%s/ manually.\n", addresses[0])
			}
		}
		return starter(ln, handler)
	}
}

func writeUsage(w io.Writer) {
	fmt.Fprint(w, `mdview serves a Markdown directory on a local HTTP server.
Markdown files ending in .md are rendered as sanitized HTML previews.

Usage:
  mdview [--addr ADDR | --tailscale] [--port PORT] [--open] [PATH]
  mdview --version
  mdview --help

Examples:
  mdview
  mdview docs
  mdview --open docs
  mdview --port 8080 README-assets
  mdview --tailscale --port 8080 docs
  mdview --tailscale --no-qr docs
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

func writeStartup(stdout io.Writer, root string, access string, addresses []string) {
	writeStartupWithOptions(stdout, root, access, addresses, startupOptions{})
}

type startupOptions struct {
	showQR bool
}

func writeStartupWithOptions(stdout io.Writer, root string, access string, addresses []string, opts startupOptions) {
	fmt.Fprintf(stdout, "Serving %s\n", root)
	if access != "" {
		fmt.Fprintf(stdout, "Access: %s\n", access)
	}
	for _, addr := range addresses {
		fmt.Fprintf(stdout, "URL: http://%s/\n", addr)
	}
	if opts.showQR && len(addresses) > 0 {
		code, err := terminalQRCode("http://" + addresses[0] + "/")
		if err == nil {
			fmt.Fprint(stdout, "QR:\n", code)
		}
	}
}

func terminalQRCode(content string) (string, error) {
	code, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", err
	}

	const quietZone = 2
	bitmap := code.Bitmap()
	size := len(bitmap) + 2*quietZone

	var b strings.Builder
	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			dark := false
			bitmapRow := row - quietZone
			bitmapCol := col - quietZone
			if bitmapRow >= 0 && bitmapRow < len(bitmap) && bitmapCol >= 0 && bitmapCol < len(bitmap[bitmapRow]) {
				dark = bitmap[bitmapRow][bitmapCol]
			}
			if dark {
				b.WriteString("##")
			} else {
				b.WriteString("  ")
			}
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func writerIsTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func openPrimaryURL(openBrowser BrowserOpener, addresses []string) error {
	if openBrowser == nil {
		openBrowser = defaultBrowserOpener
	}
	if len(addresses) == 0 {
		return fmt.Errorf("no URL available")
	}
	return openBrowser("http://" + addresses[0] + "/")
}
