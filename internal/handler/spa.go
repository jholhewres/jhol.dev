package handler

import (
	"io/fs"
	"net/http"
	"strings"
)

type SPA struct {
	fileFS  fs.FS
	handler http.Handler
}

func NewSPA(fsys fs.FS) *SPA {
	return &SPA{
		fileFS:  fsys,
		handler: http.FileServer(http.FS(fsys)),
	}
}

func (s *SPA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to serve the exact file
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Check if the file exists in the embedded FS
	f, err := s.fileFS.Open(path)
	if err == nil {
		f.Close()
		s.handler.ServeHTTP(w, r)
		return
	}

	// File not found — serve index.html for SPA client-side routing
	indexFile, err := fs.ReadFile(s.fileFS, "index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexFile)
}
