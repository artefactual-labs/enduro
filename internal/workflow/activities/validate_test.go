package activities

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	bagit "github.com/artefactual-labs/bagit-gython"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/temporal"
)

func TestStagedBagValidation(t *testing.T) {
	v, err := bagit.NewValidator(bagit.WithTempCacheDir())
	assert.NilError(t, err)
	t.Cleanup(func() { assert.NilError(t, v.Close()) })
	activity := NewValidateTransferActivity(v)
	check := func(path string, required bool) error {
		return activity.Execute(t.Context(), &ValidateTransferActivityParams{
			Path: path, CheckBagCompleteness: true, RequireBag: required,
		})
	}

	t.Run("only explicitly bagged inputs require a declaration", func(t *testing.T) {
		path := t.TempDir()
		assert.NilError(t, check(path, false))
		err := check(path, true)
		assert.ErrorContains(t, err, "bagit.txt is missing")
		assert.Assert(t, temporal.NonRetryableError(err))

		assert.NilError(t, os.Mkdir(filepath.Join(path, "bagit.txt"), 0o755))
		err = check(path, false)
		assert.ErrorContains(t, err, "bagit.txt must be a regular file")
		assert.Assert(t, temporal.NonRetryableError(err))
	})

	t.Run("unavailable staging and runner faults can be retried", func(t *testing.T) {
		err := check(filepath.Join(t.TempDir(), "missing"), false)
		assert.ErrorContains(t, err, "open staged transfer")
		assert.Assert(t, !temporal.NonRetryableError(err))

		path := t.TempDir()
		writeStagedBag(t, path)
		closed, err := bagit.NewValidator(bagit.WithDeferredRuntime())
		assert.NilError(t, err)
		assert.NilError(t, closed.Close())
		err = NewValidateTransferActivity(closed).Execute(t.Context(), &ValidateTransferActivityParams{
			Path: path, CheckBagCompleteness: true,
		})
		assert.ErrorIs(t, err, bagit.ErrClosed)
		assert.Assert(t, !temporal.NonRetryableError(err))
	})

	t.Run("the declaration must resolve inside the transfer", func(t *testing.T) {
		dir := t.TempDir()
		writeStagedBag(t, dir)
		marker := filepath.Join(dir, "bagit.txt")
		assert.NilError(t, os.Rename(marker, filepath.Join(dir, "declaration")))
		assert.NilError(t, os.Symlink("declaration", marker))
		assert.NilError(t, check(dir, true))
		assert.NilError(t, os.Remove(marker))
		assert.NilError(t, os.Symlink(filepath.Join(t.TempDir(), "declaration"), marker))
		err := check(dir, false)
		assert.ErrorContains(t, err, `"bagit.txt" must resolve inside the transfer`)
		assert.Assert(t, temporal.NonRetryableError(err))
	})

	for _, name := range []string{"data", "manifest-sha256.txt", "data/a file.txt"} {
		t.Run("external link at "+name, func(t *testing.T) {
			dir := t.TempDir()
			writeStagedBag(t, dir)
			link := filepath.Join(dir, name)
			target := filepath.Join(t.TempDir(), "original")
			assert.NilError(t, os.Rename(link, target))
			assert.NilError(t, os.Symlink(target, link))
			err := check(dir, false)
			assert.ErrorContains(t, err, fmt.Sprintf("%q must resolve inside the transfer", name))
			assert.Assert(t, temporal.NonRetryableError(err))
		})
	}

	t.Run("special files are rejected before invoking the runner", func(t *testing.T) {
		dir := t.TempDir()
		writeStagedBag(t, dir)
		pipe := filepath.Join(dir, "pipe")
		assert.NilError(t, syscall.Mkfifo(pipe, 0o600))
		marker := filepath.Join(dir, "bag-info.txt")
		assert.NilError(t, os.Symlink("pipe", marker))
		// A closed validator makes a regression fail promptly instead of
		// letting a runner block on the pipe.
		closed, err := bagit.NewValidator(bagit.WithDeferredRuntime())
		assert.NilError(t, err)
		assert.NilError(t, closed.Close())
		for _, name := range []string{"bag-info.txt", "pipe"} {
			err := NewValidateTransferActivity(closed).Execute(t.Context(), &ValidateTransferActivityParams{
				Path: dir, CheckBagCompleteness: true,
			})
			assert.ErrorContains(t, err, fmt.Sprintf("unsupported file type in transfer: %q", name))
			assert.Assert(t, temporal.NonRetryableError(err))
			assert.NilError(t, os.Remove(filepath.Join(dir, name)))
		}
		assert.NilError(t, check(dir, true))
	})

	t.Run("retained hidden files and optional bag-info", func(t *testing.T) {
		path := t.TempDir()
		writeStagedBag(t, path)
		assert.NilError(t, check(path, false))
		// Membership is intact even though the checksum no longer matches.
		assert.NilError(t, os.WriteFile(filepath.Join(path, "data", "a file.txt"), []byte("edit"), 0o600))
		assert.NilError(t, check(path, true))
	})

	t.Run("large Oxum failure reports counts and bounded examples", func(t *testing.T) {
		path := t.TempDir()
		writeStagedBag(t, path)
		var manifest strings.Builder
		sum := sha256.Sum256(nil)
		for i := range 1000 {
			fmt.Fprintf(&manifest, "%x  data/missing-%04d\n", sum, i)
		}
		assert.NilError(t, os.WriteFile(filepath.Join(path, "manifest-sha256.txt"), []byte(manifest.String()), 0o600))
		assert.NilError(t, os.WriteFile(filepath.Join(path, "bag-info.txt"), []byte("Payload-Oxum: 0.1000\n"), 0o600))
		err := check(path, true)
		assert.ErrorContains(t, err, "1000 missing, 2 unexpected")
		assert.Assert(t, temporal.NonRetryableError(err))
		samples := strings.Count(err.Error(), "[missing ") + strings.Count(err.Error(), "[unexpected ")
		assert.Equal(t, samples, 10)
	})
}

func writeStagedBag(t *testing.T, dir string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	files := map[string]string{
		"bagit.txt":           "BagIt-Version: 1.0\nTag-File-Character-Encoding: UTF-8\n",
		"data/a file.txt":     "good",
		"data/.DS_Store":      "",
		"manifest-sha256.txt": fmt.Sprintf("%x  data/a file.txt\n%x  data/.DS_Store\n", sha256.Sum256([]byte("good")), sha256.Sum256(nil)),
	}
	for name, data := range files {
		assert.NilError(t, os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600))
	}
}
