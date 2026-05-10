package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/i9wa4/markdown-remote-viewer/internal/assets"
)

type Server struct {
	root    string
	handler http.Handler
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

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("GET /", http.FileServer(http.Dir(absRoot)))

	return &Server{root: absRoot, handler: mux}, nil
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) Handler() http.Handler {
	return s.handler
}
