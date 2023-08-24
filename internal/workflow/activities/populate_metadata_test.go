package activities

import (
	"testing"

	"github.com/go-logr/logr"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestPopulateMetadataActivity(t *testing.T) {
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})
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
