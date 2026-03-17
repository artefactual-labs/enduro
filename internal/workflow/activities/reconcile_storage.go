package activities

import (
	"context"
	"errors"
	"time"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/reconciliation"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// ReconcileStorageActivity looks up a known AIP in Storage Service and
// classifies the observed state against the pipeline's recovery policy.
//
// It is intentionally limited to reporting what Storage Service currently says
// about the package. The workflow layer remains responsible for deciding what
// to do with that result, for example whether to finish successfully, fall back
// to a fresh reprocess, or stop with an error.
type ReconcileStorageActivity struct {
	pipelineRegistry  *pipeline.Registry
	getStoragePackage func(context.Context, *pipeline.Pipeline, string) (*pipeline.StoragePackage, error)
}

func NewReconcileStorageActivity(pipelineRegistry *pipeline.Registry) *ReconcileStorageActivity {
	return &ReconcileStorageActivity{
		pipelineRegistry: pipelineRegistry,
		getStoragePackage: func(ctx context.Context, p *pipeline.Pipeline, aipID string) (*pipeline.StoragePackage, error) {
			return p.GetStoragePackageForReconciliation(ctx, aipID, p.Config().Recovery)
		},
	}
}

type ReconcileStorageActivityParams struct {
	PipelineName string
	AIPID        string
}

type ReconcileStorageActivityResponse struct {
	// Classification is the detailed reconciliation outcome such as
	// `not_found`, `local_complete`, or `replicated_partial`.
	Classification reconciliation.Classification
	// Status is the coarser collection-facing status derived from Classification.
	Status reconciliation.Status
	// PrimaryExists reports whether Storage Service currently confirms that the
	// primary package exists for the requested AIP.
	PrimaryExists bool
	// StorageComplete reports whether the package satisfies the configured
	// recovery policy, including any required locations.
	StorageComplete bool
	// AIPStoredAt is the time Storage Service associates with the primary package
	// when that information is available.
	AIPStoredAt *string
	// CompletedAt is the time Enduro treats as final storage completion for the
	// configured policy. For replicated policies this may be later than
	// AIPStoredAt.
	CompletedAt *string
}

func (a *ReconcileStorageActivity) Execute(ctx context.Context, params *ReconcileStorageActivityParams) (*ReconcileStorageActivityResponse, error) {
	p, err := a.pipelineRegistry.ByName(params.PipelineName)
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}

	pkg, err := a.getStoragePackage(ctx, p, params.AIPID)
	if err != nil {
		if errors.Is(err, pipeline.ErrStoragePackageNotFound) {
			result := reconciliation.ClassifyStorage(p.Config().Recovery, nil)
			return formatReconciliationResponse(result), nil
		}
		return nil, err
	}

	result := reconciliation.ClassifyStorage(p.Config().Recovery, pkg)
	return formatReconciliationResponse(result), nil
}

func formatReconciliationResponse(result reconciliation.Result) *ReconcileStorageActivityResponse {
	return &ReconcileStorageActivityResponse{
		Classification:  result.Classification,
		Status:          result.Status,
		PrimaryExists:   result.PrimaryExists,
		StorageComplete: result.StorageComplete,
		AIPStoredAt:     formatOptionalTimePtr(result.AIPStoredAt),
		CompletedAt:     formatOptionalTimePtr(result.CompletedAt),
	}
}

func formatOptionalTimePtr(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
