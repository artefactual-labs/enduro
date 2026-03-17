package reconciliation

import (
	"slices"
	"time"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

// Classification is the normalized storage outcome produced by the
// reconciliation classifier.
type Classification string

const (
	// ClassificationNotFound means Storage Service has no package for the AIP
	// UUID, so a full reprocess may still be appropriate.
	ClassificationNotFound Classification = "not_found"
	// ClassificationLocalComplete means the primary AIP exists and no required
	// replica locations are configured for the pipeline.
	ClassificationLocalComplete Classification = "local_complete"
	// ClassificationReplicatedPartial means the primary AIP exists but at least
	// one configured required replica is still missing or lacks a stored date.
	ClassificationReplicatedPartial Classification = "replicated_partial"
	// ClassificationReplicatedComplete means the primary AIP exists and every
	// configured required replica has a stored date.
	ClassificationReplicatedComplete Classification = "replicated_complete"
	// ClassificationIndeterminate means Storage Service returned a package but
	// Enduro could not safely determine storage completion from it.
	ClassificationIndeterminate Classification = "indeterminate"
)

// Status is the persisted reconciliation status that collection records can
// store independently of the top-level workflow status.
type Status string

const (
	// StatusPending means reconciliation has not yet produced a storage-complete
	// or storage-incomplete conclusion for the collection.
	StatusPending Status = "pending"
	// StatusPartial means the AIP exists but configured storage requirements are
	// not yet satisfied.
	StatusPartial Status = "partial"
	// StatusComplete means the configured storage requirements are satisfied.
	StatusComplete Status = "complete"
	// StatusUnknown means Enduro could not safely classify the current Storage
	// Service state.
	StatusUnknown Status = "unknown"
)

// Result summarizes what Enduro concluded from Storage Service for a single
// AIP lookup.
type Result struct {
	// Classification is the detailed storage outcome used by workflow logic.
	Classification Classification
	// Status is the coarser persisted reconciliation state derived from the
	// classification.
	Status Status
	// PrimaryExists reports whether Storage Service confirms that the primary AIP
	// exists.
	PrimaryExists bool
	// StorageComplete reports whether the configured storage obligations are
	// satisfied.
	StorageComplete bool
	// AIPStoredAt is the primary AIP stored timestamp, if known.
	AIPStoredAt *time.Time
	// CompletedAt is the final storage completion time, if known. For local
	// storage this is the same as AIPStoredAt; for replicated storage this is the
	// latest required replica stored time.
	CompletedAt *time.Time
}

// ClassifyStorage translates Storage Service observations into Enduro's
// recovery model. The policy decision is intentionally small:
// when no required locations are configured, the primary AIP is the completion
// target; otherwise every configured location must have a replica with a
// concrete stored date before storage is considered complete.
func ClassifyStorage(cfg pipeline.RecoveryConfig, pkg *pipeline.StoragePackage) Result {
	if pkg == nil {
		return Result{
			Classification: ClassificationNotFound,
			Status:         StatusPending,
		}
	}

	if pkg.StoredDate == nil {
		return Result{
			Classification: ClassificationIndeterminate,
			Status:         StatusUnknown,
			PrimaryExists:  true,
		}
	}

	result := Result{
		PrimaryExists: true,
		AIPStoredAt:   pkg.StoredDate,
	}

	if len(cfg.RequiredLocations) == 0 {
		result.Classification = ClassificationLocalComplete
		result.Status = StatusComplete
		result.StorageComplete = true
		result.CompletedAt = pkg.StoredDate
		return result
	}

	latestReplicaStoredAt := *pkg.StoredDate
	for _, locationID := range cfg.RequiredLocations {
		replicaStoredAt, ok := requiredLocationStoredAt(pkg, locationID)
		if !ok {
			result.Classification = ClassificationReplicatedPartial
			result.Status = StatusPartial
			return result
		}

		if replicaStoredAt.After(latestReplicaStoredAt) {
			latestReplicaStoredAt = *replicaStoredAt
		}
	}

	result.Classification = ClassificationReplicatedComplete
	result.Status = StatusComplete
	result.StorageComplete = true
	result.CompletedAt = &latestReplicaStoredAt

	return result
}

func requiredLocationStoredAt(pkg *pipeline.StoragePackage, locationID string) (*time.Time, bool) {
	// The primary package location is reported separately from replicas, so it
	// must satisfy required locations too when operators include it explicitly.
	if pkg.CurrentLocation.UUID == locationID && pkg.StoredDate != nil {
		storedAt := *pkg.StoredDate
		return &storedAt, true
	}

	for _, replica := range pkg.Replicas {
		if replica.CurrentLocation.UUID != locationID {
			continue
		}

		// The stored date is the durable signal that the replica reached the
		// required location; status text alone is too deployment-specific.
		if replica.StoredDate == nil {
			return nil, false
		}

		storedAt := *replica.StoredDate
		return &storedAt, true
	}

	return nil, false
}

// IsKnownStatus reports whether a string matches one of the reconciliation
// statuses currently persisted by Enduro.
func IsKnownStatus(value string) bool {
	return slices.Contains([]Status{StatusPending, StatusPartial, StatusComplete, StatusUnknown}, Status(value))
}
