package fsutil

import (
	"errors"
	"os"

	"github.com/otiai10/copy"
)

// Used for testing.
var Renamer = os.Rename

// Move moves files or directories. It copies the contents when the move op
// failes because source and destination do not share the same filesystem.
func Move(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return errors.New("destination already exists")
	}

	// Move when possible.
	err := Renamer(src, dst)
	if err == nil {
		return nil
	}

	// Copy and delete otherwise.
	lerr, _ := err.(*os.LinkError)
	if lerr.Err.Error() == "invalid cross-device link" {
		err := copy.Copy(src, dst, copy.Options{
			Sync:        true,
			OnDirExists: func(src, dst string) copy.DirExistsAction { return copy.Untouchable },
		})
		if err != nil {
			return err
		}
		return os.RemoveAll(src)
	}

	return err
}
