package publisher

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

func cleanRelPath(relPath string) (string, error) {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	if relPath == "" {
		return "", errors.New("transfer publisher requires a non-empty transfer path")
	}
	if path.IsAbs(relPath) {
		return "", fmt.Errorf("transfer publisher requires a relative transfer path: %q", relPath)
	}

	clean := path.Clean(relPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("transfer publisher requires a path within the transfer source: %q", relPath)
	}

	return clean, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
