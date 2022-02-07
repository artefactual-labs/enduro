package fsutil_test

import (
	"errors"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/fsutil"
)

var Renamer = os.Rename

var dirOpts = []fs.PathOp{
	fs.WithDir(
		"child1",
		fs.WithFile(
			"foo.txt",
			"foo",
		),
	),
	fs.WithDir(
		"child2",
		fs.WithFile(
			"bar.txt",
			"bar",
		),
	),
}

func TestMove(t *testing.T) {
	t.Parallel()

	t.Run("It fails if destination already exists", func(t *testing.T) {
		t.Parallel()

		tmpDir := fs.NewDir(t, "enduro")
		fs.Apply(t, tmpDir, fs.WithFile("foobar.txt", ""))
		fs.Apply(t, tmpDir, fs.WithFile("barfoo.txt", ""))

		src := tmpDir.Join("foobar.txt")
		dst := tmpDir.Join("barfoo.txt")
		err := fsutil.Move(src, dst)

		assert.Error(t, err, "destination already exists")
	})

	t.Run("It moves files", func(t *testing.T) {
		t.Parallel()

		tmpDir := fs.NewDir(t, "enduro")
		fs.Apply(t, tmpDir, fs.WithFile("foobar.txt", ""))

		src := tmpDir.Join("foobar.txt")
		dst := tmpDir.Join("barfoo.txt")
		err := fsutil.Move(src, dst)

		assert.NilError(t, err)

		_, err = os.Stat(src)
		assert.ErrorIs(t, err, os.ErrNotExist)

		_, err = os.Stat(dst)
		assert.NilError(t, err)
	})

	t.Run("It moves directories", func(t *testing.T) {
		t.Parallel()

		tmpSrc := fs.NewDir(t, "enduro", dirOpts...)
		src := tmpSrc.Path()
		srcManifest := fs.ManifestFromDir(t, src)
		tmpDst := fs.NewDir(t, "enduro")
		dst := tmpDst.Join("nested")

		err := fsutil.Move(src, dst)

		assert.NilError(t, err)
		_, err = os.Stat(src)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Assert(t, fs.Equal(dst, srcManifest))
	})

	t.Run("It copies directories when using different filesystems", func(t *testing.T) {
		fsutil.Renamer = func(src, dst string) error {
			return &os.LinkError{
				Op:  "rename",
				Old: src,
				New: dst,
				Err: errors.New("invalid cross-device link"),
			}
		}
		t.Cleanup(func() {
			fsutil.Renamer = os.Rename
		})

		tmpSrc := fs.NewDir(t, "enduro", dirOpts...)
		src := tmpSrc.Path()
		srcManifest := fs.ManifestFromDir(t, src)
		tmpDst := fs.NewDir(t, "enduro")
		dst := tmpDst.Join("nested")

		err := fsutil.Move(src, dst)

		assert.NilError(t, err)
		_, err = os.Stat(src)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Assert(t, fs.Equal(dst, srcManifest))
	})
}
