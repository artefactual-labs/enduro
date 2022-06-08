package workflow

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
)

type ProcessingWorkflowTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env *temporalsdk_testsuite.TestWorkflowEnvironment

	// Each test registers the workflow with a different name to avoid dups.
	workflow *ProcessingWorkflow
}

func (s *ProcessingWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	ctrl := gomock.NewController(s.T())
	s.workflow = NewProcessingWorkflow(
		logr.Discard(),
		collectionfake.NewMockService(ctrl),
		watcherfake.NewMockService(ctrl),
	)
}

func (s *ProcessingWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestProcessingWorkflow(t *testing.T) {
	suite.Run(t, new(ProcessingWorkflowTestSuite))
}
