package activities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestPopulateMetadataActivity(t *testing.T) {
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{}, nil, nil)
	activity := NewPopulateMetadataActivity(pipelineRegistry)
	tempdir := fs.NewDir(t, "enduro")

	expected := fs.Expected(
		t,
		fs.WithDir(
			"metadata",
			fs.WithFile(
				"metadata.csv",
				"parts,dc.identifier\nobjects,12345\n",
				fs.WithMode(0o664),
			),
		),
	)

	s := temporalsdk_testsuite.WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	_, err := env.ExecuteActivity(activity.Execute, &PopulateMetadataActivityParams{
		Identifier: "12345",
		Path:       tempdir.Path(),
	})

	assert.NilError(t, err)
	assert.Assert(t, fs.Equal(tempdir.Path(), expected))
}

func TestPopulateMetadataConfinesWrites(t *testing.T) {
	for _, linkDir := range []bool{true, false} {
		dir, external := t.TempDir(), t.TempDir()
		target := filepath.Join(external, "metadata.csv")
		assert.NilError(t, os.WriteFile(target, []byte("original"), 0o600))
		if linkDir {
			assert.NilError(t, os.Symlink(external, filepath.Join(dir, "metadata")))
		} else {
			assert.NilError(t, os.Mkdir(filepath.Join(dir, "metadata"), 0o755))
			assert.NilError(t, os.Symlink(target, filepath.Join(dir, "metadata", "metadata.csv")))
		}
		err := NewPopulateMetadataActivity(nil).Execute(t.Context(), &PopulateMetadataActivityParams{Path: dir, Identifier: "12345"})
		assert.Assert(t, err != nil)
		content, err := os.ReadFile(target)
		assert.NilError(t, err)
		assert.Equal(t, string(content), "original")
	}
}
