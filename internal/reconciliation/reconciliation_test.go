package reconciliation

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestClassifyStorage(t *testing.T) {
	t.Parallel()

	primaryStoredAt := time.Date(2026, 3, 17, 7, 0, 0, 0, time.UTC)
	replicaStoredAt := time.Date(2026, 3, 17, 8, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		cfg  pipeline.RecoveryConfig
		pkg  *pipeline.StoragePackage
		want Result
	}{
		"Nil package is not found": {
			cfg: pipeline.RecoveryConfig{},
			pkg: nil,
			want: Result{
				Classification: ClassificationNotFound,
				Status:         StatusPending,
			},
		},
		"Primary without stored date is indeterminate": {
			cfg: pipeline.RecoveryConfig{},
			pkg: &pipeline.StoragePackage{},
			want: Result{
				Classification: ClassificationIndeterminate,
				Status:         StatusUnknown,
				PrimaryExists:  true,
			},
		},
		"Local storage completes from primary stored date": {
			cfg: pipeline.RecoveryConfig{},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
			},
			want: Result{
				Classification:  ClassificationLocalComplete,
				Status:          StatusComplete,
				PrimaryExists:   true,
				StorageComplete: true,
				AIPStoredAt:     &primaryStoredAt,
				CompletedAt:     &primaryStoredAt,
			},
		},
		"Replicated storage is partial when a required location is missing": {
			cfg: pipeline.RecoveryConfig{
				RequiredLocations: []string{"loc-1", "loc-2"},
			},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
				Replicas: []pipeline.StorageReplica{
					{
						StoredDate: &replicaStoredAt,
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-1",
						},
					},
				},
			},
			want: Result{
				Classification: ClassificationReplicatedPartial,
				Status:         StatusPartial,
				PrimaryExists:  true,
				AIPStoredAt:    &primaryStoredAt,
			},
		},
		"Replicated storage is partial when a required replica has no stored date": {
			cfg: pipeline.RecoveryConfig{
				RequiredLocations: []string{"loc-1"},
			},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
				Replicas: []pipeline.StorageReplica{
					{
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-1",
						},
					},
				},
			},
			want: Result{
				Classification: ClassificationReplicatedPartial,
				Status:         StatusPartial,
				PrimaryExists:  true,
				AIPStoredAt:    &primaryStoredAt,
			},
		},
		"Replicated storage is partial when required locations are configured but replicas are empty": {
			cfg: pipeline.RecoveryConfig{
				RequiredLocations: []string{"loc-1"},
			},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
			},
			want: Result{
				Classification: ClassificationReplicatedPartial,
				Status:         StatusPartial,
				PrimaryExists:  true,
				AIPStoredAt:    &primaryStoredAt,
			},
		},
		"Replicated storage completes from latest required replica date": {
			cfg: pipeline.RecoveryConfig{
				RequiredLocations: []string{"loc-1", "loc-2"},
			},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
				Replicas: []pipeline.StorageReplica{
					{
						StoredDate: &primaryStoredAt,
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-1",
						},
					},
					{
						StoredDate: &replicaStoredAt,
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-2",
						},
					},
				},
			},
			want: Result{
				Classification:  ClassificationReplicatedComplete,
				Status:          StatusComplete,
				PrimaryExists:   true,
				StorageComplete: true,
				AIPStoredAt:     &primaryStoredAt,
				CompletedAt:     &replicaStoredAt,
			},
		},
		"Replicated storage completes when three required locations are present": {
			cfg: pipeline.RecoveryConfig{
				RequiredLocations: []string{"loc-1", "loc-2", "loc-3"},
			},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
				Replicas: []pipeline.StorageReplica{
					{
						StoredDate: &primaryStoredAt,
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-1",
						},
					},
					{
						StoredDate: &replicaStoredAt,
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-2",
						},
					},
					{
						StoredDate: func() *time.Time {
							later := replicaStoredAt.Add(time.Hour)
							return &later
						}(),
						CurrentLocation: pipeline.StorageLocation{
							UUID: "loc-3",
						},
					},
				},
			},
			want: Result{
				Classification:  ClassificationReplicatedComplete,
				Status:          StatusComplete,
				PrimaryExists:   true,
				StorageComplete: true,
				AIPStoredAt:     &primaryStoredAt,
				CompletedAt: func() *time.Time {
					later := replicaStoredAt.Add(time.Hour)
					return &later
				}(),
			},
		},
		"Required locations may include the primary package location": {
			cfg: pipeline.RecoveryConfig{
				RequiredLocations: []string{"primary-loc", "replica-loc"},
			},
			pkg: &pipeline.StoragePackage{
				StoredDate: &primaryStoredAt,
				CurrentLocation: pipeline.StorageLocation{
					UUID: "primary-loc",
				},
				Replicas: []pipeline.StorageReplica{
					{
						StoredDate: &replicaStoredAt,
						CurrentLocation: pipeline.StorageLocation{
							UUID: "replica-loc",
						},
					},
				},
			},
			want: Result{
				Classification:  ClassificationReplicatedComplete,
				Status:          StatusComplete,
				PrimaryExists:   true,
				StorageComplete: true,
				AIPStoredAt:     &primaryStoredAt,
				CompletedAt:     &replicaStoredAt,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyStorage(tc.cfg, tc.pkg)
			assert.DeepEqual(t, got, tc.want)
		})
	}
}
