package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
)

type ProcessingWorkflowTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env *temporalsdk_testsuite.TestWorkflowEnvironment

	hooks *hooks.Hooks

	// Each test registers the workflow with a different name to avoid dups.
	workflow *ProcessingWorkflow
}

func (s *ProcessingWorkflowTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.env = s.NewTestWorkflowEnvironment()
	s.hooks = buildHooks(s.T(), ctrl)
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
}

func (s *ProcessingWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

// Workflow ignores an error in parseName when NHA hooks are disabled.
func (s *ProcessingWorkflowTestSuite) TestParseErrorIsIgnored() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	// Collection is persisted.
	s.env.OnActivity(createPackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint(12345), nil).Once()

	// parseName is executed with errors.
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(nil, errors.New("parse error")).Once()

	// loadConfig is executed (workflow continued), returning an error.
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("pipeline is unavailable")).Once()

	// Defer updates the package with the error status before returning.
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	retentionPeriod := time.Second
	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:     0,
		WatcherName:      "watcher",
		PipelineName:     "pipeline",
		RetentionPeriod:  &retentionPeriod,
		StripTopLevelDir: true,
		Key:              "key",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "pipeline is unavailable")
}

// Workflow does not ignore an error in parseName when NHA hooks are enabled.
func (s *ProcessingWorkflowTestSuite) TestParseError() {
	s.hooks.Hooks["hari"]["disabled"] = false
	s.hooks.Hooks["prod"]["disabled"] = false

	// Collection is persisted.
	s.env.OnActivity(createPackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint(12345), nil).Once()

	// parseName is executed, inject error.
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(nil, errors.New("parse error")).Once()

	// Defer updates the package with the error status before returning.
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	retentionPeriod := time.Second
	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:     0,
		WatcherName:      "watcher",
		PipelineName:     "pipeline",
		RetentionPeriod:  &retentionPeriod,
		StripTopLevelDir: true,
		Key:              "key",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "parse error")
}

func TestProcessingWorkflow(t *testing.T) {
	suite.Run(t, new(ProcessingWorkflowTestSuite))
}

func buildHooks(t *testing.T, ctrl *gomock.Controller) *hooks.Hooks {
	t.Helper()

	return hooks.NewHooks(
		map[string]map[string]interface{}{
			"prod": {"disabled": "false"},
			"hari": {"disabled": "false"},
		},
	)
}

func TestTransferInfoProcessingConfiguration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		tinfo            TransferInfo
		processingConfig string
	}{
		{
			tinfo: TransferInfo{
				ProcessingConfig: "automated",
				PipelineConfig: &pipeline.Config{
					ProcessingConfig: "default",
				},
			},
			processingConfig: "automated",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "automated",
				PipelineConfig:   nil,
			},
			processingConfig: "automated",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "",
				PipelineConfig:   nil,
			},
			processingConfig: "",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "",
				PipelineConfig: &pipeline.Config{
					ProcessingConfig: "default",
				},
			},
			processingConfig: "default",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "",
				PipelineConfig: &pipeline.Config{
					ProcessingConfig: "",
				},
			},
			processingConfig: "",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()

			result := tc.tinfo.ProcessingConfiguration()
			if have, want := result, tc.processingConfig; have != want {
				t.Errorf("tinfo.ProcessingConfiguration() returned %s; expected %s", have, want)
			}
		})
	}
}
