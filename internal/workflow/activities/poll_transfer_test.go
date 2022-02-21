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

func TestPollTransferActivity(t *testing.T) {
	t.Run("Fails when the pipeline isn't found", func(t *testing.T) {
		manager := newManager(t, nil)
		activity := NewPollTransferActivity(manager)
		manager.Pipelines, _ = pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)

		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{})

		assert.Assert(t, is.Nil(future))
		assert.Error(t, err, "non retryable error")
	})

	t.Run("Identifies packages that completed processing successfully", func(t *testing.T) {
		ctx := context.Background()
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"sip_uuid": "734abbaf-4e2f-4a68-938c-c8ff6420e525",
				"status": "COMPLETE"
			}`))
		})
		activity := NewPollTransferActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

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
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		activity := NewPollTransferActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{
			PipelineName: "am",
			TransferID:   "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
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
		activity := NewPollTransferActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

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
		manager := newManager(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			clock.(clockwork.FakeClock).Advance(time.Minute)
			w.WriteHeader(http.StatusBadGateway)
		})
		activity := NewPollTransferActivity(manager)

		s := cadencesdk_testsuite.WorkflowTestSuite{}
		env := s.NewTestActivityEnvironment()
		env.RegisterActivity(activity.Execute)
		env.SetWorkerOptions(cadencesdk_worker.Options{BackgroundActivityContext: ctx})

		future, err := env.ExecuteActivity(activity.Execute, &PollTransferActivityParams{
			PipelineName: "am",
			TransferID:   "cbc4b312-b076-4ff7-b67b-b6850f2b4486",
		})

		assert.Assert(t, is.Nil(future))
		assert.Error(t, err, "non retryable error")
		assert.Equal(t, backoffStrategy.(*ZeroBackOff).hits, 6)
	})
}
