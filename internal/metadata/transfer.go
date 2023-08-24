package metadata

import (
	"path/filepath"
	"strings"
)

type TransferName struct {
	DCIdentifier string // (MoMA: ComponentNumber)
	ComponentID  string // (MoMA: ComponentID)
	Accession    string // (MoMA: ObjectID)
}

const separator = "---"

func FromTransferName(name string, isDir bool) TransferName {
	if name == "" {
		return TransferName{}
	}

	// Remove file extension.
	if !isDir {
		name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	}

	parts := strings.Split(name, separator)
	switch len(parts) {
	case 3:
		return TransferName{DCIdentifier: parts[0], ComponentID: parts[1], Accession: parts[2]}
	case 2:
		return TransferName{DCIdentifier: parts[0], ComponentID: parts[1]}
	default:
		return TransferName{DCIdentifier: name}
	}
}
