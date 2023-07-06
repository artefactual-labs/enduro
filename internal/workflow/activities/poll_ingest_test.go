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

func TestPollIngestActivity(t *testing.T) {
	t.Run("Fails when the pipeline isn't found", func(t *testing.T) {
		pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})
		activity := NewPollIngestActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{})

		assert.Assert(t, future == nil)
		assert.ErrorContains(t, err, "unknown pipeline")
		assert.Assert(t, temporal.NonRetryableError(err) == true)
	})

	t.Run("Identifies packages that completed processing successfully", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		clock = clockwork.NewFakeClockAt(now)
		pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"sip_uuid": "734abbaf-4e2f-4a68-938c-c8ff6420e525",
				"status": "COMPLETE"
			}`))
		})
		activity := NewPollIngestActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		var storedAt time.Time
		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{
			PipelineName: "am",
			SIPID:        "734abbaf-4e2f-4a68-938c-c8ff6420e525",
		})
		future.Get(&storedAt)

		assert.NilError(t, err)
		assert.Equal(t, storedAt.Sub(now).String(), "0s")
	})

	t.Run("Abandons when a non-retryable error is detected", func(t *testing.T) {
		ctx := context.Background()
		pipelineRegistry := newPipelineRegistry(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		activity := NewPollIngestActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{
			PipelineName: "am",
			SIPID:        "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})

		assert.Assert(t, future == nil)
		assert.ErrorContains(t, err, "error checking ingest status: server error")
		assert.Assert(t, temporal.NonRetryableError(err) == true)
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
		activity := NewPollIngestActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		_, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{
			PipelineName: "am",
			SIPID:        "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})

		assert.NilError(t, err)
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
		activity := NewPollIngestActivity(pipelineRegistry)

		s := temporalsdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(temporalsdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{
			PipelineName: "am",
			SIPID:        "734abbaf-4e2f-4a68-938c-c8ff6420e525",
		})

		assert.Assert(t, future == nil)
		assert.Assert(t, temporal.NonRetryableError(err) == true)
		assert.ErrorContains(t, err, "error checking ingest status")
		assert.Equal(t, backoffStrategy.(*ZeroBackOff).hits, 6)
	})
}
