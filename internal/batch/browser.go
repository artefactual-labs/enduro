package batch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
)

const browserEntryLimit = 1000

func (s *batchImpl) Browse(ctx context.Context, payload *goabatch.BrowsePayload) (*goabatch.BatchBrowseResult, error) {
	if s.browserRoot == "" {
		return nil, goabatch.MakeNotAvailable(errors.New("batch browser is not configured"))
	}

	apiPath, rootPath, err := normalizeBrowserPath(payload.Path)
	if err != nil {
		return nil, goabatch.MakeNotValid(err)
	}

	root, err := os.OpenRoot(s.browserRoot)
	if err != nil {
		return nil, goabatch.MakeNotAvailable(errors.New("batch browser root is unavailable"))
	}
	defer root.Close()

	dir, err := root.Open(rootPath)
	if err != nil {
		return nil, goabatch.MakeNotValid(errors.New("directory is unavailable"))
	}
	defer dir.Close()

	info, err := dir.Stat()
	if err != nil {
		return nil, goabatch.MakeNotValid(errors.New("directory is unavailable"))
	}
	if !info.IsDir() {
		return nil, goabatch.MakeNotValid(errors.New("path is not a directory"))
	}

	entries, truncated, err := readBrowserEntries(ctx, dir, apiPath, s.browserRoot)
	if err != nil {
		return nil, goabatch.MakeNotValid(errors.New("directory is unavailable"))
	}

	return &goabatch.BatchBrowseResult{
		Path:         apiPath,
		AbsolutePath: filepath.Join(s.browserRoot, filepath.FromSlash(apiPath)),
		Entries:      entries,
		Truncated:    truncated,
	}, nil
}

func normalizeBrowserPath(value *string) (apiPath, rootPath string, err error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "", ".", nil
	}

	raw := strings.TrimSpace(*value)
	if path.IsAbs(raw) || filepath.IsAbs(raw) || strings.Contains(raw, `\`) {
		return "", "", fmt.Errorf("path must be relative to the batch browser root")
	}

	cleaned := path.Clean(raw)
	if cleaned == "." {
		return "", ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", "", fmt.Errorf("path must stay within the batch browser root")
	}

	return cleaned, filepath.FromSlash(cleaned), nil
}

func readBrowserEntries(ctx context.Context, dir *os.File, parentPath, browserRoot string) ([]*goabatch.BatchBrowseEntry, bool, error) {
	entries := make([]*goabatch.BatchBrowseEntry, 0)
	truncated := false

	for {
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		default:
		}

		batch, err := dir.ReadDir(100)
		if err != nil && err != io.EOF {
			return nil, false, err
		}
		for _, entry := range batch {
			if !entry.IsDir() {
				continue
			}
			if len(entries) >= browserEntryLimit {
				truncated = true
				break
			}
			childPath := path.Join(parentPath, entry.Name())
			if parentPath == "" {
				childPath = entry.Name()
			}
			item := &goabatch.BatchBrowseEntry{
				Name:         entry.Name(),
				Path:         childPath,
				AbsolutePath: filepath.Join(browserRoot, filepath.FromSlash(childPath)),
			}
			if info, err := entry.Info(); err == nil {
				item.ModifiedAt = formatBrowserTime(info.ModTime())
			}
			entries = append(entries, item)
		}
		if err == io.EOF || truncated {
			break
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return entries, truncated, nil
}

func formatBrowserTime(value time.Time) *string {
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
