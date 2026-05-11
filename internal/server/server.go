package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/i9wa4/markdown-remote-viewer/internal/assets"
)

type Server struct {
	root    string
	handler http.Handler
	files   http.Handler
}

func New(root string) (*Server, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory: %s", root)
	}

	static, err := fs.Sub(assets.FS, "static")
	if err != nil {
		return nil, fmt.Errorf("load static assets: %w", err)
	}

	srv := &Server{
		root:  absRoot,
		files: http.FileServer(http.Dir(absRoot)),
	}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /", srv.serveRoot)
	srv.handler = mux

	return srv, nil
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) serveRoot(w http.ResponseWriter, r *http.Request) {
	filePath, ok := s.markdownFilePath(r.URL.Path)
	if !ok {
		s.files.ServeHTTP(w, r)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}

	source, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	page, err := renderMarkdownPage(path.Base(r.URL.Path), source)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: http: https:; script-src 'none'; object-src 'none'; base-uri 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(page)
}

func (s *Server) markdownFilePath(urlPath string) (string, bool) {
	cleaned := path.Clean("/" + urlPath)
	if !strings.HasSuffix(strings.ToLower(cleaned), ".md") {
		return "", false
	}

	rel := strings.TrimPrefix(cleaned, "/")
	if rel == "" || rel == "." {
		return "", false
	}

	filePath := filepath.Join(s.root, filepath.FromSlash(rel))
	relToRoot, err := filepath.Rel(s.root, filePath)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filePath, true
}
