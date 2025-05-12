package ui

import (
	"embed"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Assets contains the web front-end static assets.
//
// Some valid accessors:
//
//	dist [dir]
//	dist/assets [dir]
//	dist/index.html [file]
//	dist/favicon.ico [file]
//
//go:embed dist/*
var Assets embed.FS

func SPAHandler() http.HandlerFunc {
	const (
		root   = "/"
		prefix = "dist"
		index  = "/index.html"
	)
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the absolute path to prevent directory traversal.
		path, err := filepath.Abs(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Serve index file when the path is empty.
		if path == root {
			path = index
		}

		// Prepend dist prefix.
		path = filepath.Join(prefix, path)

		// Open and convert to io.ReadSeeker.
		file, err := Assets.Open(path)
		frs, ok := file.(io.ReadSeeker)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.ServeContent(w, r, path, time.Now(), frs)
	}
}
