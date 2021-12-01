package main

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
)

const (
	FixityWorkflowName              = "fixity"
	FixityListPackagesActivityName  = "fixity:list-packages"
	FixityCheckPackagesActivityName = "fixity:check-packages"
	FixityReportActivityName        = "fixity:report"
	FixityTaskListName              = "global"
)

// registerFixityWorkflowActivities registers fixity-related workflows and activities.
func registerFixityWorkflowActivities(w worker.Worker) {
	// Register fixity workflow.
	w.RegisterWorkflowWithOptions(
		FixityWorkflow,
		workflow.RegisterOptions{Name: FixityWorkflowName},
	)

	// Register fixity activities.
	w.RegisterActivityWithOptions(
		FixityListPackagesActivity,
		activity.RegisterOptions{Name: FixityListPackagesActivityName},
	)
	w.RegisterActivityWithOptions(
		FixityCheckPackagesActivity,
		activity.RegisterOptions{Name: FixityCheckPackagesActivityName},
	)
	w.RegisterActivityWithOptions(
		FixityReportActivity,
		activity.RegisterOptions{Name: FixityReportActivityName},
	)
}

func FixityWorkflow(ctx workflow.Context) error {
	var packages []Package
	var results PackageFixityResults

	// List packages.
	{
		activityOptions := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			TaskList:               FixityTaskListName,
			ScheduleToStartTimeout: time.Second * 10,
			StartToCloseTimeout:    time.Minute,
		})

		future := workflow.ExecuteActivity(activityOptions, FixityListPackagesActivity)

		err := future.Get(ctx, &packages)
		if err != nil {
			return err
		}
	}

	// Check packages.
	{
		activityOptions := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			TaskList:               FixityTaskListName,
			ScheduleToStartTimeout: time.Second * 10,
			StartToCloseTimeout:    time.Minute,
		})

		future := workflow.ExecuteActivity(activityOptions, FixityCheckPackagesActivityName, packages)

		err := future.Get(ctx, &results)
		if err != nil {
			return err
		}
	}

	// Report.
	{
		if results.Passed() {
			return nil
		}

		activityOptions := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			TaskList:               FixityTaskListName,
			ScheduleToStartTimeout: time.Second * 10,
			StartToCloseTimeout:    time.Minute,
		})

		future := workflow.ExecuteActivity(activityOptions, FixityReportActivityName, results)

		err := future.Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func FixityListPackagesActivity(ctx context.Context) ([]Package, error) {
	list := []Package{
		{ID: "81338b38-c918-499a-b02d-783b7febb4ee"},
		{ID: "78a1dfd2-8361-4496-92d5-5a0c50f308a8"},
		{ID: "10602c4a-7941-4db7-9f72-de713942c5a4"},
		{ID: "77dc93ce-e3f5-4eb1-8e78-87e1d1e116df"},
		{ID: "3fcd7d38-c024-40f1-9840-cad20cef75b9", Corrupted: true},
	}

	return list, nil
}

func FixityCheckPackagesActivity(ctx context.Context, packages []Package) (PackageFixityResults, error) {
	results := PackageFixityResults{}

	for _, pkg := range packages {
		results[pkg.ID] = pkg.Fixity()
	}

	return results, nil
}

func FixityReportActivity(ctx context.Context, results PackageFixityResults) error {
	for identifier, passed := range results {
		if passed {
			continue
		}

		fmt.Println("-------------------------------------------------------------------------------------------------")
		fmt.Printf(">>> Package %s did not pass the fixity check!\n", identifier)
		fmt.Println("-------------------------------------------------------------------------------------------------")
	}

	return nil
}

type Package struct {
	ID        string
	Corrupted bool
}

func (p Package) Fixity() bool {
	// Expensive cryptographic computation to perform integrity check.
	time.Sleep(time.Second * 3)

	return !p.Corrupted
}

type PackageFixityResults map[string]bool

func (r PackageFixityResults) Passed() bool {
	for _, passed := range r {
		if !passed {
			return false
		}
	}

	return true
}
