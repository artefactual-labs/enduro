package activities

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestBundleActivity(t *testing.T) {
	t.Parallel()

	t.Run("Excludes hidden files", func(t *testing.T) {
		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(
			t, "enduro",
			fs.WithDir(
				"transfer",
				fs.WithFile("foobar.txt", "Hello world!\n"),
				fs.WithFile(".hidden", ""),
			),
		)

		transferSourceDir := fs.NewDir(t, "enduro")

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			TempFile:           transferDir.Join("transfer"),
			ExcludeHiddenFiles: true,
			IsDir:              true,
			TransferDir:        transferSourceDir.Path(),
			Key:                "transfer",
		})
		assert.NilError(t, err)

		// Capture final destination directory within the transfer source
		// directory, i.e. Copy method uses a random name.
		items, err := os.ReadDir(transferSourceDir.Path())
		assert.NilError(t, err)
		destDir := filepath.Join(transferSourceDir.Path(), items[0].Name())

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.DeepEqual(t, res, res)
		assert.Assert(t,
			fs.Equal(
				destDir,
				fs.Expected(t,
					// .hidden is not expected because ExcludeHiddenFiles is enabled.
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.MatchAnyFileMode,
				),
			),
		)
	})

	t.Run("Remove hidden files when BatchDir is a subfolder of the TransferDir", func(t *testing.T) {
		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro",
			fs.WithDir("batch-folder",
				fs.WithDir(
					"sip",
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.WithFile(".hidden", ""),
				),
			),
		)
		batchDir := transferDir.Join("batch-folder")
		sipSourceDir := transferDir.Join("batch-folder", "sip")

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			ExcludeHiddenFiles: true,
			IsDir:              true,
			TransferDir:        transferDir.Path(),
			BatchDir:           batchDir,
			Key:                "sip",
		})
		assert.NilError(t, err)

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.Assert(t,
			fs.Equal(
				sipSourceDir,
				fs.Expected(t,
					// .hidden is not expected because ExcludeHiddenFiles is enabled.
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.MatchAnyFileMode,
				),
			),
		)
		assert.DeepEqual(t, res.FullPath, sipSourceDir)
		rePath, err := filepath.Rel(transferDir.Path(), sipSourceDir)
		assert.NilError(t, err)
		assert.DeepEqual(t, res.RelPath, rePath)
	})
}

func TestUnbag(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		syscall.Umask(2)
	}

	tempdir := fs.NewDir(
		t,
		"enduro",
		fs.WithDir(
			"data",
			fs.WithFile(
				"foobar.txt",
				"Hello world!\n",
			),
		),
		fs.WithFile(
			"bag-info.txt",
			`Bag-Software-Agent: bagit.py v1.8.1 <https://github.com/LibraryOfCongress/bagit-python>
Bagging-Date: 2022-01-17
Payload-Oxum: 13.1
`),
		fs.WithFile(
			"bagit.txt",
			`BagIt-Version: 0.97
Tag-File-Character-Encoding: UTF-8
`),
		fs.WithFile(
			"manifest-sha256.txt",
			`0ba904eae8773b70c75333db4de2f3ac45a8ad4ddba1b242f0b3cfc199391dd8  data/foobar.txt
`),
		fs.WithFile(
			"tagmanifest-sha256.txt",
			`f00810e0385d173109b2b3121ec29a16e0737b4ac9e30f2eaa9d3aac813aacae manifest-sha256.txt
2c3cbd8249b6f98b6385d02246c6b9b4e6c2e78267cc1a6fe5d2e954b017fda2 bag-info.txt
e91f941be5973ff71f1dccbdd1a32d598881893a7f21be516aca743da38b1689 bagit.txt
`),
	)

	expected := fs.Expected(
		t,
		fs.WithFile(
			"foobar.txt",
			"Hello world!\n",
		),
		fs.WithDir(
			"metadata",
			fs.WithMode(0o775),
			fs.WithFile(
				"checksum.sha256",
				"0ba904eae8773b70c75333db4de2f3ac45a8ad4ddba1b242f0b3cfc199391dd8  ../objects/foobar.txt\n",
				fs.WithMode(0o664),
			),
			fs.WithDir(
				"submissionDocumentation",
				fs.WithMode(0o775),
				fs.WithFile(
					"bag-info.txt",
					`Bag-Software-Agent: bagit.py v1.8.1 <https://github.com/LibraryOfCongress/bagit-python>
Bagging-Date: 2022-01-17
Payload-Oxum: 13.1
`),
				fs.WithFile(
					"bagit.txt",
					`BagIt-Version: 0.97
Tag-File-Character-Encoding: UTF-8
`),
				fs.WithFile(
					"manifest-sha256.txt",
					`0ba904eae8773b70c75333db4de2f3ac45a8ad4ddba1b242f0b3cfc199391dd8  data/foobar.txt
`),
				fs.WithFile(
					"tagmanifest-sha256.txt",
					`f00810e0385d173109b2b3121ec29a16e0737b4ac9e30f2eaa9d3aac813aacae manifest-sha256.txt
2c3cbd8249b6f98b6385d02246c6b9b4e6c2e78267cc1a6fe5d2e954b017fda2 bag-info.txt
e91f941be5973ff71f1dccbdd1a32d598881893a7f21be516aca743da38b1689 bagit.txt
`),
			),
		),
	)

	path := tempdir.Path()
	err := unbag(path)

	assert.NilError(t, err)
	assert.Assert(t, fs.Equal(path, expected))
}
