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

const contentSecurityPolicy = "default-src 'self'; img-src 'self' data: http: https:; script-src 'none'; object-src 'none'; base-uri 'none'"

type Server struct {
	root     string
	rootReal string
	handler  http.Handler
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
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve root symlinks: %w", err)
	}

	static, err := fs.Sub(assets.FS, "static")
	if err != nil {
		return nil, fmt.Errorf("load static assets: %w", err)
	}

	srv := &Server{
		root:     absRoot,
		rootReal: realRoot,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /", srv.serveRoot)
	srv.handler = withSecurityHeaders(mux)

	return srv, nil
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) directoryIndexRequestContained(urlPath string) bool {
	filePath, ok := s.safeFilePath(urlPath)
	if !ok {
		return true
	}
	return s.directoryIndexContained(filePath)
}

func (s *Server) directoryIndexContained(realPath string) bool {
	info, err := os.Stat(realPath)
	if err != nil || !info.IsDir() {
		return true
	}
	realIndex, err := filepath.EvalSymlinks(filepath.Join(realPath, "index.html"))
	if err != nil {
		return os.IsNotExist(err)
	}
	rel, err := filepath.Rel(s.rootReal, realIndex)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.directoryIndexRequestContained(r.URL.Path) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		s.handler.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) serveRoot(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(strings.ToLower(path.Clean("/"+r.URL.Path)), ".md") {
		s.serveStatic(w, r)
		return
	}

	filePath, ok := s.safeFilePath(r.URL.Path)
	if !ok {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
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
	_, _ = w.Write(page)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	filePath, ok := s.safeFilePath(r.URL.Path)
	if !ok {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, filePath)
}

func (s *Server) safeFilePath(urlPath string) (string, bool) {
	cleaned := path.Clean("/" + urlPath)
	rel := strings.TrimPrefix(cleaned, "/")
	if rel == "" {
		rel = "."
	}

	filePath := filepath.Join(s.root, filepath.FromSlash(rel))
	relToRoot, err := filepath.Rel(s.root, filePath)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", false
	}

	realPath, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return filePath, true
		}
		return "", false
	}
	realRel, err := filepath.Rel(s.rootReal, realPath)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return realPath, true
}
