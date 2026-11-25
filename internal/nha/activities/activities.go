// Package activities implements workflow activities specific to Norway Health
// Authority.
package activities

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/artefactual-labs/enduro/internal/bagit"
)

var (
	UpdateHARIActivityName             = "update-hari-activity"
	UpdateProductionSystemActivityName = "update-production-system-activity"
)

type receiptPayload struct {
	*os.Root
	bag       bool
	manifests []string // Original payload manifests from older conversions.
	layout    *bagit.ConvertedLayout
}

func openReceiptPayload(path string, layout *bagit.ConvertedLayout) (*receiptPayload, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	if layout != nil {
		return &receiptPayload{Root: root, bag: true, layout: layout}, nil
	}
	marker, err := root.Stat("bagit.txt")
	if err == nil {
		defer root.Close()
		if !marker.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid BagIt declaration in %q", path)
		}
		data, err := root.OpenRoot("data")
		if err != nil {
			return nil, err
		}
		return &receiptPayload{Root: data, bag: true}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		root.Close()
		return nil, err
	}

	const docs = "metadata/submissionDocumentation/"
	if _, err := root.Stat(docs + "bagit.txt"); errors.Is(err, os.ErrNotExist) {
		return &receiptPayload{Root: root}, nil
	} else if err != nil {
		root.Close()
		return nil, err
	}
	manifests, err := fs.Glob(root.FS(), docs+"manifest-*.txt")
	if err != nil {
		root.Close()
		return nil, err
	}
	if len(manifests) == 0 {
		// An ordinary transfer may include a declaration in its documentation.
		return &receiptPayload{Root: root}, nil
	}
	return &receiptPayload{Root: root, bag: true, manifests: manifests}, nil
}

// includes checks the staged filename and its possible original payload paths.
func (p *receiptPayload) includes(staged string, original ...string) (bool, error) {
	if p.layout != nil {
		name, _, _ := strings.Cut(staged, "/")
		return slices.Contains(p.layout.PayloadRoots, name), nil
	}
	// Older activity results lack layout information. Require agreement among
	// the archived manifests so one extra document cannot authorize a tag.
	for _, manifest := range p.manifests {
		ok, err := p.manifestIncludes(manifest, original)
		if !ok || err != nil {
			return false, err
		}
	}
	return true, nil
}

func (p *receiptPayload) manifestIncludes(manifest string, paths []string) (bool, error) {
	file, err := p.Open(manifest)
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.IndexAny(line, " \t")
		if i < 0 {
			continue
		}
		name := strings.TrimLeft(line[i:], " \t")
		for _, path := range paths {
			if name == "data/"+path {
				return true, nil
			}
		}
	}
	return false, scanner.Err()
}
