package activities

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	bagit "github.com/artefactual-labs/bagit-gython"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/watcher"
)

func TestBundleActivity(t *testing.T) {
	t.Parallel()

	t.Run("a staging directory inside the input is cleaned up", func(t *testing.T) {
		source := t.TempDir()
		destination := filepath.Join(source, "transfers")
		assert.NilError(t, os.Mkdir(destination, 0o700))
		_, err := NewBundleActivity(nil, nil).Execute(t.Context(), &BundleActivityParams{
			TempFile: source, TransferDir: destination, IsDir: true,
		})
		assert.ErrorContains(t, err, "cannot stage a transfer inside its source directory")
		items, err := os.ReadDir(destination)
		assert.NilError(t, err)
		assert.Equal(t, len(items), 0)
	})

	t.Run("Excludes hidden files", func(t *testing.T) {
		activity := NewBundleActivity(nil, nil)
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
		activity := NewBundleActivity(nil, nil)
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
		activity := NewBundleActivity(nil, nil)
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
		activity := NewBundleActivity(nil, nil)
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
		activity := NewBundleActivity(nil, nil)
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

	t.Run("Bundles batch file when BatchDir is outside TransferDir", func(t *testing.T) {
		activity := NewBundleActivity(nil, nil)
		ts := &temporalsdk_testsuite.WorkflowTestSuite{}
		env := ts.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		transferDir := fs.NewDir(t, "enduro")
		batchDir := fs.NewDir(t, "batch",
			fs.WithFile("transfer1.zip", "zip contents\n"),
		)

		fut, err := env.ExecuteActivity(activity.Execute, &BundleActivityParams{
			IsDir:       false,
			TransferDir: transferDir.Path(),
			BatchDir:    batchDir.Path(),
			Key:         "transfer1.zip",
		})
		assert.NilError(t, err)

		res := BundleActivityResult{}
		assert.NilError(t, fut.Get(&res))
		assert.Assert(t, strings.HasPrefix(res.FullPath, transferDir.Path()+string(os.PathSeparator)))
		assert.Equal(t, res.FullPathBeforeStrip, res.FullPath)
		assert.Assert(t, strings.HasPrefix(res.RelPath, "c"+string(os.PathSeparator)))

		data, err := os.ReadFile(filepath.Join(res.FullPath, "objects", "transfer1.zip"))
		assert.NilError(t, err)
		assert.Equal(t, string(data), "zip contents\n")

		original, err := os.ReadFile(batchDir.Join("transfer1.zip"))
		assert.NilError(t, err)
		assert.Equal(t, string(original), "zip contents\n")
	})

	t.Run("Copies external batch when unrelated transfer contents are unreadable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("chmod permissions are not portable on Windows")
		}

		activity := NewBundleActivity(nil, nil)
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
		activity := NewBundleActivity(nil, nil)
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

func TestBundleWatchedDirectory(t *testing.T) {
	watched := t.TempDir()
	nested := filepath.Join(watched, "transfer", "nested")
	assert.NilError(t, os.MkdirAll(nested, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(nested, "a file.txt"), []byte("good"), 0o600))
	assert.NilError(t, os.Symlink("a file.txt", filepath.Join(nested, "alias.txt")))
	assert.NilError(t, os.Symlink("transfer", filepath.Join(watched, "inside")))
	assert.NilError(t, os.Symlink(t.TempDir(), filepath.Join(watched, "outside")))
	svc, err := watcher.New(t.Context(), &watcher.Config{
		Filesystem: []*watcher.FilesystemConfig{{Name: "drop", Path: watched}},
	})
	assert.NilError(t, err)
	activity := NewBundleActivity(svc, nil)
	params := &BundleActivityParams{WatcherName: "drop", Key: "inside", IsDir: true, TransferDir: t.TempDir()}
	result, err := activity.Execute(t.Context(), params)
	assert.NilError(t, err)
	content, err := os.ReadFile(filepath.Join(result.FullPath, "nested", "a file.txt"))
	assert.NilError(t, err)
	assert.Equal(t, string(content), "good")

	target, err := os.Readlink(filepath.Join(result.FullPath, "nested", "alias.txt"))
	assert.NilError(t, err)
	assert.Equal(t, target, "a file.txt")

	assert.NilError(t, os.Symlink(t.TempDir(), filepath.Join(nested, "external")))
	params.TransferDir = t.TempDir()
	_, err = activity.Execute(t.Context(), params)
	assert.ErrorContains(t, err, "must resolve inside the transfer")
	entries, err := os.ReadDir(params.TransferDir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)

	params.Key = "outside"
	_, err = activity.Execute(t.Context(), params)
	assert.ErrorContains(t, err, "open transfer in watcher directory")
	assert.Assert(t, temporal.NonRetryableError(err))
}

func TestBundleFailedBagConversion(t *testing.T) {
	watched := t.TempDir()
	bag := filepath.Join(watched, "bag")
	assert.NilError(t, os.MkdirAll(filepath.Join(bag, "data"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(bag, "data", "payload.txt"), []byte("Hello world!\n"), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(bag, "bagit.txt"), []byte("BagIt-Version: 0.97\nTag-File-Character-Encoding: UTF-8\n"), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(bag, "manifest-sha256.txt"), []byte("0ba904eae8773b70c75333db4de2f3ac45a8ad4ddba1b242f0b3cfc199391dd8  data/payload.txt\n"), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(bag, "bag-info.txt"), []byte("Payload-Oxum: 13.1\n"), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(bag, "metadata"), []byte("occupied"), 0o600))
	svc, err := watcher.New(t.Context(), &watcher.Config{
		Filesystem: []*watcher.FilesystemConfig{{Name: "drop", Path: watched}},
	})
	assert.NilError(t, err)
	// Exercise conversion itself even when the separate preflight is disabled.
	activity := NewBundleActivity(svc, nil)
	params := &BundleActivityParams{
		WatcherName: "drop", Key: "bag", TransferDir: t.TempDir(),
		IsDir: true, Unbag: true,
	}

	_, err = activity.Execute(t.Context(), params)
	assert.ErrorContains(t, err, "metadata")
	assert.Assert(t, temporal.NonRetryableError(err))
	entries, err := os.ReadDir(params.TransferDir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)
	content, err := os.ReadFile(filepath.Join(bag, "data", "payload.txt"))
	assert.NilError(t, err)
	assert.Equal(t, string(content), "Hello world!\n")

	// Failed conversion also leaves an in-place batch source untouched.
	outside := t.TempDir()
	assert.NilError(t, os.Remove(filepath.Join(bag, "metadata")))
	assert.NilError(t, os.Symlink(outside, filepath.Join(bag, "metadata")))
	params.BatchDir = watched
	params.TransferDir = watched
	_, err = activity.Execute(t.Context(), params)
	assert.ErrorContains(t, err, "metadata")
	entries, err = os.ReadDir(outside)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)
	_, err = os.Stat(filepath.Join(bag, "bagit.txt"))
	assert.NilError(t, err)
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
`,
		),
		fs.WithFile(
			"bagit.txt",
			`BagIt-Version: 0.97
Tag-File-Character-Encoding: UTF-8
`,
		),
		fs.WithFile(
			"manifest-sha256.txt",
			`0ba904eae8773b70c75333db4de2f3ac45a8ad4ddba1b242f0b3cfc199391dd8  data/foobar.txt
`,
		),
		fs.WithFile(
			"tagmanifest-sha256.txt",
			`f00810e0385d173109b2b3121ec29a16e0737b4ac9e30f2eaa9d3aac813aacae manifest-sha256.txt
2c3cbd8249b6f98b6385d02246c6b9b4e6c2e78267cc1a6fe5d2e954b017fda2 bag-info.txt
e91f941be5973ff71f1dccbdd1a32d598881893a7f21be516aca743da38b1689 bagit.txt
`,
		),
	)

	expected := fs.Expected(
		t,
		fs.WithDir("objects", fs.WithFile("foobar.txt", "Hello world!\n"), fs.MatchAnyFileMode),
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
`,
				),
				fs.WithFile(
					"bagit.txt",
					`BagIt-Version: 0.97
Tag-File-Character-Encoding: UTF-8
`,
				),
				fs.WithFile(
					"manifest-sha256.txt",
					`0ba904eae8773b70c75333db4de2f3ac45a8ad4ddba1b242f0b3cfc199391dd8  data/foobar.txt
`,
				),
				fs.WithFile(
					"tagmanifest-sha256.txt",
					`f00810e0385d173109b2b3121ec29a16e0737b4ac9e30f2eaa9d3aac813aacae manifest-sha256.txt
2c3cbd8249b6f98b6385d02246c6b9b4e6c2e78267cc1a6fe5d2e954b017fda2 bag-info.txt
e91f941be5973ff71f1dccbdd1a32d598881893a7f21be516aca743da38b1689 bagit.txt
`,
				),
			),
		),
	)

	path := tempdir.Path()
	converted, err := unbag(path)
	assert.Check(t, converted != nil)

	assert.NilError(t, err)
	assert.Assert(t, fs.Equal(path, expected))
}

func TestConvertBagKeepsMetadataBesideObjects(t *testing.T) {
	for _, journal := range []string{"DPJ/journal.xml", "objects/DPJ/journal.xml"} {
		t.Run(journal, func(t *testing.T) {
			const metadata = "metadata/identifiers.json"
			xml, identifiers := "<journal/>", "[]"
			manifest := fmt.Sprintf("%x  data/%s\n%x  data/%s\n", sha256.Sum256([]byte(xml)), journal, sha256.Sum256([]byte(identifiers)), metadata)
			dir := fs.NewDir(t, "bag",
				fs.WithFile("bagit.txt", "BagIt-Version: 0.97\nTag-File-Character-Encoding: UTF-8\n"),
				fs.WithFile("bag-info.txt", fmt.Sprintf("Payload-Oxum: %d.2\n", len(xml)+len(identifiers))),
				fs.WithFile("manifest-sha256.txt", manifest),
				fs.WithDir("data/"+filepath.Dir(journal), fs.WithFile("journal.xml", xml)),
				fs.WithDir("data/metadata", fs.WithFile("identifiers.json", identifiers)),
			)
			defer dir.Remove()
			result, err := NewBundleActivity(nil, nil).Execute(t.Context(), &BundleActivityParams{
				TempFile: dir.Path(), TransferDir: t.TempDir(), IsDir: true, Unbag: true,
			})
			assert.NilError(t, err)
			assert.Assert(t, result.Unbagged)
			for name, want := range map[string]string{
				"objects/DPJ/journal.xml": xml, metadata: identifiers,
				"metadata/checksum.sha256": fmt.Sprintf("%x  ../objects/DPJ/journal.xml\n%x  ../metadata/identifiers.json\n", sha256.Sum256([]byte(xml)), sha256.Sum256([]byte(identifiers))),
			} {
				data, err := os.ReadFile(filepath.Join(result.FullPath, name))
				assert.NilError(t, err)
				assert.Equal(t, string(data), want)
			}
		})
	}
}

func TestInPlaceBagConversionCanBeRetried(t *testing.T) {
	const content = "hello"
	source := fs.NewDir(t, "batch", fs.WithDir("bag",
		fs.WithFile("bagit.txt", "BagIt-Version: 0.97\nTag-File-Character-Encoding: UTF-8\n"),
		fs.WithFile("bag-info.txt", "Payload-Oxum: 5.1\n"),
		fs.WithFile("manifest-sha256.txt", fmt.Sprintf("%x  data/content\n", sha256.Sum256([]byte(content)))),
		fs.WithDir("data", fs.WithFile("content", content)),
	))
	defer source.Remove()
	bags, err := bagit.NewValidator(bagit.WithTempCacheDir())
	assert.NilError(t, err)
	defer bags.Close()
	activity := NewBundleActivity(nil, bags)
	params := &BundleActivityParams{
		BatchDir: source.Path(), TransferDir: source.Path(), Key: "bag",
		IsDir: true, Unbag: true, RequireBag: true, CheckBagBeforeUnbag: true,
	}
	for range 2 {
		result, err := activity.Execute(t.Context(), params)
		assert.NilError(t, err)
		assert.Assert(t, result.Unbagged)
		assert.Assert(t, result.FullPath != source.Join("bag"))
		assert.Equal(t, result.FullPath, source.Join(result.RelPath))
		data, err := os.ReadFile(filepath.Join(result.FullPath, "objects", "content"))
		assert.NilError(t, err)
		assert.Equal(t, string(data), content)
		assert.NilError(t, os.RemoveAll(result.FullPathBeforeStrip))
	}
	data, err := os.ReadFile(source.Join("bag", "data", "content"))
	assert.NilError(t, err)
	assert.Equal(t, string(data), content)
}

func TestConversionRejectsLinkBrokenByMetadataMove(t *testing.T) {
	const target = "metadata/record.txt"
	content := strings.Repeat("x", len(target))
	hash := sha256.Sum256([]byte(content))
	source := fs.NewDir(t, "bag",
		fs.WithFile("bagit.txt", "BagIt-Version: 0.97\nTag-File-Character-Encoding: UTF-8\n"),
		fs.WithFile("bag-info.txt", fmt.Sprintf("Payload-Oxum: %d.2\n", 2*len(content))),
		fs.WithFile("manifest-sha256.txt", fmt.Sprintf("%x  data/alias\n%x  data/%s\n", hash, hash, target)),
		fs.WithDir("data/metadata", fs.WithFile("record.txt", content)),
	)
	defer source.Remove()
	assert.NilError(t, os.Symlink(target, source.Join("data", "alias")))
	stage := t.TempDir()
	bags, err := bagit.NewValidator(bagit.WithTempCacheDir())
	assert.NilError(t, err)
	defer bags.Close()
	_, err = NewBundleActivity(nil, bags).Execute(t.Context(), &BundleActivityParams{
		TempFile: source.Path(), TransferDir: stage, IsDir: true, Unbag: true, CheckBagBeforeUnbag: true, RequireBag: true,
	})
	assert.ErrorContains(t, err, "conversion changes the target of payload link")
	data, err := os.ReadFile(source.Join("data", "alias"))
	assert.NilError(t, err)
	assert.Equal(t, string(data), content)
	entries, err := os.ReadDir(stage)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)
}
