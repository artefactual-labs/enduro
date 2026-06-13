package activities

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		assert.Equal(t, res.FullPath, sipSourceDir)
		rePath, err := filepath.Rel(transferDir.Path(), sipSourceDir)
		assert.NilError(t, err)
		assert.Equal(t, res.RelPath, rePath)
	})

	t.Run("Reuses batch transfer when BatchDir is a filesystem alias of TransferDir", func(t *testing.T) {
		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro",
			fs.WithDir("Enduro_Testing",
				fs.WithDir("general_pipeline",
					fs.WithDir(
						"sip",
						fs.WithFile("foobar.txt", "Hello world!\n"),
					),
				),
			),
		)

		aliasRoot := filepath.Join(t.TempDir(), "transfer_source")
		if err := os.Symlink(transferDir.Path(), aliasRoot); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}

		batchDir := filepath.Join(aliasRoot, "Enduro_Testing", "general_pipeline")
		sipSourceDir := filepath.Join(batchDir, "sip")

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			IsDir:       true,
			TransferDir: transferDir.Path(),
			BatchDir:    batchDir,
			Key:         "sip",
		})
		assert.NilError(t, err)

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.Equal(t, res.FullPath, sipSourceDir)
		assert.Equal(t, res.FullPathBeforeStrip, "")
		assert.Equal(t, res.RelPath, filepath.Join("Enduro_Testing", "general_pipeline", "sip"))

		items, err := os.ReadDir(transferDir.Path())
		assert.NilError(t, err)
		for _, item := range items {
			assert.Assert(t, !strings.HasPrefix(item.Name(), "enduro"))
		}
	})

	t.Run("Reuses batch transfer when BatchDir is a filesystem alias of a TransferDir subdirectory", func(t *testing.T) {
		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro",
			fs.WithDir("Enduro_Testing",
				fs.WithDir("general_pipeline",
					fs.WithDir(
						"sip",
						fs.WithFile("foobar.txt", "Hello world!\n"),
					),
				),
			),
		)

		aliasRoot := filepath.Join(t.TempDir(), "incoming")
		if err := os.Symlink(transferDir.Join("Enduro_Testing"), aliasRoot); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}

		batchDir := filepath.Join(aliasRoot, "general_pipeline")
		sipSourceDir := filepath.Join(batchDir, "sip")

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			IsDir:       true,
			TransferDir: transferDir.Path(),
			BatchDir:    batchDir,
			Key:         "sip",
		})
		assert.NilError(t, err)

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.Equal(t, res.FullPath, sipSourceDir)
		assert.Equal(t, res.FullPathBeforeStrip, "")
		assert.Equal(t, res.RelPath, filepath.Join("Enduro_Testing", "general_pipeline", "sip"))

		items, err := os.ReadDir(transferDir.Path())
		assert.NilError(t, err)
		for _, item := range items {
			assert.Assert(t, !strings.HasPrefix(item.Name(), "enduro"))
		}
	})

	t.Run("Copies batch transfer when BatchDir is outside TransferDir", func(t *testing.T) {
		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro")
		batchDir := fs.NewDir(t, "batch",
			fs.WithDir(
				"sip",
				fs.WithFile("foobar.txt", "Hello world!\n"),
			),
		)

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			IsDir:       true,
			TransferDir: transferDir.Path(),
			BatchDir:    batchDir.Path(),
			Key:         "sip",
		})
		assert.NilError(t, err)

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.Assert(t, strings.HasPrefix(res.FullPath, transferDir.Path()+string(os.PathSeparator)))
		assert.Equal(t, res.FullPathBeforeStrip, res.FullPath)
		assert.Assert(t, strings.HasPrefix(res.RelPath, "enduro"))
		assert.Assert(t,
			fs.Equal(
				res.FullPath,
				fs.Expected(t,
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.MatchAnyFileMode,
				),
			),
		)
	})

	t.Run("Copies external batch when unrelated transfer contents are unreadable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod permissions are not portable on Windows")
		}

		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro",
			fs.WithDir("unrelated",
				fs.WithFile("foobar.txt", "Hello world!\n"),
			),
		)
		unrelatedDir := transferDir.Join("unrelated")
		assert.NilError(t, os.Chmod(unrelatedDir, 0))
		t.Cleanup(func() {
			_ = os.Chmod(unrelatedDir, 0o755)
		})
		if _, err := os.ReadDir(unrelatedDir); err == nil {
			t.Skip("chmod did not make the directory unreadable")
		}

		batchDir := fs.NewDir(t, "batch",
			fs.WithDir(
				"sip",
				fs.WithFile("foobar.txt", "Hello world!\n"),
			),
		)

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			IsDir:       true,
			TransferDir: transferDir.Path(),
			BatchDir:    batchDir.Path(),
			Key:         "sip",
		})
		assert.NilError(t, err)

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.Assert(t, strings.HasPrefix(res.FullPath, transferDir.Path()+string(os.PathSeparator)))
		assert.Equal(t, res.FullPathBeforeStrip, res.FullPath)
		assert.Assert(t, strings.HasPrefix(res.RelPath, "enduro"))
		assert.Assert(t,
			fs.Equal(
				res.FullPath,
				fs.Expected(t,
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.MatchAnyFileMode,
				),
			),
		)
	})

	t.Run("Removes hidden directories recursively", func(t *testing.T) {
		activity := NewBundleActivity()
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro",
			fs.WithDir("batch-folder",
				fs.WithDir(
					"sip",
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.WithDir(".hidden",
						fs.WithFile("secret.txt", "shh\n"),
					),
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
					fs.WithFile("foobar.txt", "Hello world!\n"),
					fs.MatchAnyFileMode,
				),
			),
		)
		assert.Equal(t, res.FullPath, sipSourceDir)
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
