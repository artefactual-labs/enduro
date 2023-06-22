/*
Package bundler provides a transfer package creator for Archivematica.

It is an alternative to TransferSession that addresses some of its major pain
points. For example, this version does not concern with submitting the package
to Archivematica and makes the package filesystem available to the user for
more flexibility to alter its contents manually.

It could eventually be included in the amclient module as its own package.
*/
package bundler

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/afero"
)

func TestNewBundlerWithTempDir(t *testing.T) {
	transferDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	b, err := NewBundlerWithTempDir(transferDir)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := afero.DirExists(b.fs, "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("dir not exists")
	}

	_ = b.Write("foobar.txt", bytes.NewReader([]byte("foooooo")))
	_ = b.Bundle()
}
