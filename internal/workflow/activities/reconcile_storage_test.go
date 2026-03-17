package activities

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/reconciliation"
)

func TestReconcileStorageActivityExecute(t *testing.T) {
	t.Parallel()

	t.Run("Maps not found into reconciliation result", func(t *testing.T) {
		t.Parallel()

		activity := newReconcileStorageActivityForTest(t, pipeline.Config{
			Name: "am",
			ID:   "pipeline-1",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		}, func(context.Context, *pipeline.Pipeline, string) (*pipeline.StoragePackage, error) {
			return nil, pipeline.ErrStoragePackageNotFound
		})

		res, err := activity.Execute(context.Background(), &ReconcileStorageActivityParams{
			PipelineName: "am",
			AIPID:        "missing",
		})

		assert.NilError(t, err)
		assert.Equal(t, res.Classification, reconciliation.ClassificationNotFound)
		assert.Equal(t, res.Status, reconciliation.StatusPending)
		assert.Equal(t, res.PrimaryExists, false)
		assert.Equal(t, res.StorageComplete, false)
		assert.Assert(t, res.AIPStoredAt == nil)
		assert.Assert(t, res.CompletedAt == nil)
	})

	t.Run("Classifies replicated completion", func(t *testing.T) {
		t.Parallel()

		activity := newReconcileStorageActivityForTest(t, pipeline.Config{
			Name: "am",
			ID:   "pipeline-1",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
				RequiredLocations:    []string{"loc-1"},
			},
			StorageServiceURL: "http://user:key@example.com",
		}, func(context.Context, *pipeline.Pipeline, string) (*pipeline.StoragePackage, error) {
			primaryStoredAt := parseTestTime(t, "2026-03-17T07:00:00Z")
			replicaStoredAt := parseTestTime(t, "2026-03-17T08:00:00Z")

			return &pipeline.StoragePackage{
				UUID:            "aip-123",
				Status:          "UPLOADED",
				StoredDate:      &primaryStoredAt,
				CurrentFullPath: "/var/aips/aip-123.7z",
				CurrentLocation: pipeline.StorageLocation{UUID: "primary-loc"},
				Replicas: []pipeline.StorageReplica{
					{
						UUID:            "rep-1",
						Status:          "UPLOADED",
						StoredDate:      &replicaStoredAt,
						CurrentFullPath: "/replicas/aip-123.7z",
						CurrentLocation: pipeline.StorageLocation{UUID: "loc-1"},
					},
				},
			}, nil
		})

		res, err := activity.Execute(context.Background(), &ReconcileStorageActivityParams{
			PipelineName: "am",
			AIPID:        "aip-123",
		})

		assert.NilError(t, err)
		assert.Equal(t, res.Classification, reconciliation.ClassificationReplicatedComplete)
		assert.Equal(t, res.Status, reconciliation.StatusComplete)
		assert.Equal(t, res.PrimaryExists, true)
		assert.Equal(t, res.StorageComplete, true)
		assert.Assert(t, res.AIPStoredAt != nil)
		assert.Assert(t, res.CompletedAt != nil)
		assert.Equal(t, *res.CompletedAt, "2026-03-17T08:00:00Z")
	})

	t.Run("Returns transport errors", func(t *testing.T) {
		t.Parallel()

		activity := newReconcileStorageActivityForTest(t, pipeline.Config{
			Name: "am",
			ID:   "pipeline-1",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		}, func(context.Context, *pipeline.Pipeline, string) (*pipeline.StoragePackage, error) {
			return nil, errors.New("boom")
		})

		res, err := activity.Execute(context.Background(), &ReconcileStorageActivityParams{
			PipelineName: "am",
			AIPID:        "aip-123",
		})

		assert.Assert(t, res == nil)
		assert.ErrorContains(t, err, "boom")
	})
}

func newReconcileStorageActivityForTest(t *testing.T, cfg pipeline.Config, getter func(context.Context, *pipeline.Pipeline, string) (*pipeline.StoragePackage, error)) *ReconcileStorageActivity {
	t.Helper()

	registry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{cfg}, nil, nil)
	assert.NilError(t, err)

	activity := NewReconcileStorageActivity(registry)
	activity.getStoragePackage = getter

	return activity
}

func parseTestTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	assert.NilError(t, err)

	return parsed
}
