package frontend

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	buildRoot = ".output/public"
	indexPath = buildRoot + "/index.html"
)

// Assets contains the web front-end static assets built by Nuxt.
//
//go:embed all:.output/public
var Assets embed.FS

// SPAHandler serves the built frontend under the provided base path.
func SPAHandler(basePath string) http.HandlerFunc {
	basePath = normalizeBasePath(basePath)

	return func(w http.ResponseWriter, r *http.Request) {
		requestPath, ok := stripBasePath(r.URL.Path, basePath)
		if !ok {
			http.NotFound(w, r)
			return
		}

		assetPath := pathToAsset(requestPath)
		contentPath := assetPath
		tryDirectoryIndex := shouldTryDirectoryIndex(requestPath)
		fallbackToIndex := shouldFallbackToIndex(requestPath)

		file, name, err := openAsset(contentPath)
		if errors.Is(err, fs.ErrNotExist) && tryDirectoryIndex {
			contentPath = path.Join(contentPath, "index.html")
			file, name, err = openAsset(contentPath)
		}
		if errors.Is(err, fs.ErrNotExist) && fallbackToIndex {
			contentPath = indexPath
			file, name, err = openAsset(contentPath)
		}
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		readSeeker, ok := file.(io.ReadSeeker)
		if !ok {
			http.Error(w, "asset is not seekable", http.StatusInternalServerError)
			return
		}

		http.ServeContent(w, r, name, time.Now(), readSeeker)
	}
}

func normalizeBasePath(basePath string) string {
	if basePath == "" || basePath == "/" {
		return "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	return strings.TrimRight(basePath, "/")
}

func stripBasePath(requestPath, basePath string) (string, bool) {
	if basePath == "/" {
		if requestPath == "" {
			return "/", true
		}
		return requestPath, true
	}

	switch {
	case requestPath == basePath:
		return "/", true
	case strings.HasPrefix(requestPath, basePath+"/"):
		return strings.TrimPrefix(requestPath, basePath), true
	default:
		return "", false
	}
}

func pathToAsset(requestPath string) string {
	cleanPath := path.Clean("/" + requestPath)
	if cleanPath == "/" {
		return indexPath
	}
	return path.Join(buildRoot, strings.TrimPrefix(cleanPath, "/"))
}

func shouldFallbackToIndex(requestPath string) bool {
	cleanPath := path.Clean("/" + requestPath)
	if cleanPath == "/" {
		return false
	}

	// Static assets should return 404 when missing.
	if strings.HasPrefix(cleanPath, "/_nuxt/") || strings.HasPrefix(cleanPath, "/_fonts/") {
		return false
	}

	// Client-side routes with no extension should resolve to index.html.
	return path.Ext(cleanPath) == ""
}

func shouldTryDirectoryIndex(requestPath string) bool {
	cleanPath := path.Clean("/" + requestPath)
	if cleanPath == "/" {
		return false
	}
	return path.Ext(cleanPath) == ""
}

func openAsset(assetPath string) (fs.File, string, error) {
	file, err := Assets.Open(assetPath)
	if err != nil {
		return nil, "", err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, "", err
	}
	if stat.IsDir() {
		file.Close()
		return nil, "", fs.ErrNotExist
	}

	return file, path.Base(assetPath), nil
}
