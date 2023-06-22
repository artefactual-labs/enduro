package activities

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TransferIdentifiers is the type that maps to the JSON-encoded document found
// in Archivematica transfers to list identifiers.
type TransferIdentifiers []TransferIdentifier

type TransferIdentifier struct {
	File        string                   `json:"file"`
	Identifiers []TransferIdentifierPair `json:"identifiers"`
}

type TransferIdentifierPair struct {
	// avleveringsidentifikator is the one we want
	Type  string `json:"identifierType"`
	Value string `json:"identifier"`
}

// readIdentifiers returns the identifiers found in the given transfer path.
func readIdentifiers(path string) (TransferIdentifiers, error) {
	identifiers := TransferIdentifiers([]TransferIdentifier{})

	blob, err := os.ReadFile(filepath.Join(path, "metadata", "identifiers.json"))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(blob, &identifiers); err != nil {
		return nil, err
	}

	return identifiers, nil
}

// readIdentifier returns the specified identifier found in the given transfer
// path. file matching uses case-insensitive suffix matching.
func readIdentifier(path string, fileSuffix string, idtype string) (string, error) {
	identifiers, err := readIdentifiers(path)
	if err != nil {
		return "", fmt.Errorf("error reading identifier: %v", err)
	}

	for _, item := range identifiers {
		if !strings.HasSuffix(strings.ToLower(item.File), strings.ToLower(fileSuffix)) {
			continue
		}
		for _, pair := range item.Identifiers {
			if pair.Type == idtype {
				return pair.Value, nil
			}
		}
	}

	return "", errors.New("error reading identifier: not found")
}
