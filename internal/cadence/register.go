package cadence

import (
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

func RegisterActivity(activityFunc interface{}, name string) {
	opts := activity.RegisterOptions{
		Name: name,
	}
	activity.RegisterWithOptions(activityFunc, opts)
}

func RegisterWorkflow(workflowFunc interface{}, name string) {
	opts := workflow.RegisterOptions{
		Name: name,
	}
	workflow.RegisterWithOptions(workflowFunc, opts)
}
