package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
)

type FixityTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestFixity(t *testing.T) {
	suite.Run(t, new(FixityTestSuite))
}

func (s *FixityTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterWorkflowWithOptions(FixityWorkflow, workflow.RegisterOptions{Name: FixityWorkflowName})
	s.env.RegisterActivityWithOptions(FixityListPackagesActivity, activity.RegisterOptions{Name: FixityListPackagesActivityName})
	s.env.RegisterActivityWithOptions(FixityCheckPackagesActivity, activity.RegisterOptions{Name: FixityCheckPackagesActivityName})
	s.env.RegisterActivityWithOptions(FixityReportActivity, activity.RegisterOptions{Name: FixityReportActivityName})
}

func (s *FixityTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

// Test_CronWorkflow runs the workflow as a child workflow using a cron
// schedule. The child workflow should, and if we wait for a couple of hours
// we should see a total of three runs.
// We will confirm that all activities are executed three times.
func (s *FixityTestSuite) Test_CronWorkflow() {
	testWorkflow := func(ctx workflow.Context) error {
		ctx1 := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			ExecutionStartToCloseTimeout: time.Minute * 10,
			CronSchedule:                 "@hourly",
		})

		// Cron workflows never stop so this future won't return.
		cronFuture := workflow.ExecuteChildWorkflow(ctx1, FixityWorkflow)

		// Wait 2 hours for the cron (cron will execute 3 times)
		workflow.Sleep(ctx, time.Hour*2)
		s.False(cronFuture.IsReady())
		return nil
	}

	s.env.RegisterWorkflow(testWorkflow)

	packages := []Package{
		{ID: "81338b38-c918-499a-b02d-783b7febb4ee"},
		{ID: "78a1dfd2-8361-4496-92d5-5a0c50f308a8"},
		{ID: "10602c4a-7941-4db7-9f72-de713942c5a4"},
		{ID: "77dc93ce-e3f5-4eb1-8e78-87e1d1e116df"},
		{ID: "3fcd7d38-c024-40f1-9840-cad20cef75b9", Corrupted: true},
	}

	results := PackageFixityResults(map[string]bool{
		"81338b38-c918-499a-b02d-783b7febb4ee": true,
		"78a1dfd2-8361-4496-92d5-5a0c50f308a8": true,
		"10602c4a-7941-4db7-9f72-de713942c5a4": true,
		"77dc93ce-e3f5-4eb1-8e78-87e1d1e116df": true,
		"3fcd7d38-c024-40f1-9840-cad20cef75b9": false,
	})

	ctx := mock.Anything

	s.env.OnActivity(FixityListPackagesActivity, ctx).Return(packages, nil).Times(3)

	s.env.OnActivity(FixityCheckPackagesActivity, ctx, packages).Return(results, nil).Times(3)

	s.env.OnActivity(FixityReportActivity, ctx, results).Return(nil).Times(3)

	s.env.ExecuteWorkflow(testWorkflow)

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}
