package activities

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/jonboulle/clockwork"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

func TestPollTransferActivity(t *testing.T) {
	t.Run("Fails when the pipeline isn't found", func(t *testing.T) {
		pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})
		activity := NewPollTransferActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{})

		assert.Assert(t, future == nil)
		assert.Assert(t, temporal.NonRetryableError(err) == true)
		assert.ErrorContains(t, err, "unknown pipeline")
	})

	t.Run("Identifies packages that completed processing successfully", func(t *testing.T) {
		ctx := context.Background()
		pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"sip_uuid": "734abbaf-4e2f-4a68-938c-c8ff6420e525",
				"status": "COMPLETE"
			}`))
		})
		activity := NewPollTransferActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		var sipID string
		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{
			PipelineName: "am",
			TransferID:   "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})
		future.Get(&sipID)

		assert.NilError(t, err)
		assert.Equal(t, sipID, "734abbaf-4e2f-4a68-938c-c8ff6420e525")
	})

	t.Run("Abandons when a non-retryable error is detected", func(t *testing.T) {
		ctx := context.Background()
		pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		activity := NewPollTransferActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{
			PipelineName: "am",
			TransferID:   "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})

		assert.Assert(t, future == nil)
		assert.Assert(t, temporal.NonRetryableError(err) == true)
		assert.ErrorContains(t, err, "error checking transfer status: server error")
	})

	t.Run("Polls until processing completes", func(t *testing.T) {
		ctx := context.Background()
		backoffStrategy = &ZeroBackOff{}
		attempts := 0
		pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts > 3 {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"sip_uuid": "734abbaf-4e2f-4a68-938c-c8ff6420e525",
					"status": "COMPLETE"
				}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"sip_uuid": "734abbaf-4e2f-4a68-938c-c8ff6420e525",
				"status": "PROCESSING"
			}`))
		})
		activity := NewPollTransferActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		var sipID string
		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{
			PipelineName: "am",
			TransferID:   "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})
		future.Get(&sipID)

		assert.NilError(t, err)
		assert.Equal(t, sipID, "734abbaf-4e2f-4a68-938c-c8ff6420e525")
		assert.Equal(t, backoffStrategy.(*ZeroBackOff).hits, 3)
	})

	t.Run("Retries on retry-able errors until the deadline is exceeded", func(t *testing.T) {
		ctx := context.Background()
		backoffStrategy = &ZeroBackOff{}
		clock = clockwork.NewFakeClock()
		attempts := 0
		pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			clock.(clockwork.FakeClock).Advance(time.Minute)
			w.WriteHeader(http.StatusBadGateway)
		})
		activity := NewPollTransferActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{
			PipelineName: "am",
			TransferID:   "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})

		assert.Assert(t, future == nil)
		assert.Assert(t, temporal.NonRetryableError(err) == true)
		assert.ErrorContains(t, err, "error checking transfer status")
		assert.Equal(t, backoffStrategy.(*ZeroBackOff).hits, 6)
	})
}
