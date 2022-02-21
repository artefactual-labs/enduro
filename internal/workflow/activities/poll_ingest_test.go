package activities

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/jonboulle/clockwork"
	cadencesdk_testsuite "go.uber.org/cadence/testsuite"
	cadencesdk_worker "go.uber.org/cadence/worker"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestPollIngestActivity(t *testing.T) {
	t.Run("Fails when the pipeline isn't found", func(t *testing.T) {
		manager := newManager(t, nil)
		activity := NewPollIngestActivity(manager)
		manager.Pipelines, _ = pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{})

		assert.Assert(t, is.Nil(future))
		assert.Error(t, err, "non retryable error")
	})

	t.Run("Identifies packages that completed processing successfully", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		clock = clockwork.NewFakeClockAt(now)
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"sip_uuid": "734abbaf-4e2f-4a68-938c-c8ff6420e525",
				"status": "COMPLETE"
			}`))
		})
		activity := NewPollIngestActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

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
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		activity := NewPollIngestActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{
			PipelineName: "am",
			SIPID:        "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})

		assert.Assert(t, is.Nil(future))
		assert.Error(t, err, "non retryable error")
	})

	t.Run("Polls until processing completes", func(t *testing.T) {
		ctx := context.Background()
		backoffStrategy = &ZeroBackOff{}
		attempts := 0
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
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
		activity := NewPollIngestActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

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
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			clock.(clockwork.FakeClock).Advance(time.Minute)
			w.WriteHeader(http.StatusBadGateway)
		})
		activity := NewPollIngestActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollIngestActivityParams{
			PipelineName: "am",
			SIPID:        "734abbaf-4e2f-4a68-938c-c8ff6420e525",
		})

		assert.Assert(t, is.Nil(future))
		assert.Error(t, err, "non retryable error")
		assert.Equal(t, backoffStrategy.(*ZeroBackOff).hits, 6)
	})
}
