// Package reconciliation models Enduro's Storage Service recovery decisions.
//
// The problem domain is a storage decision, not generic workflow state:
// when Enduro already knows an AIP UUID, what can it safely conclude from
// Storage Service about the current storage state of that AIP?
//
// This package keeps that decision logic separate from Temporal workflow code
// and separate from the pipeline's Storage Service access layer. The pipeline
// fetches package observations; this package interprets those observations
// against Enduro's configured recovery policy.
//
// The main distinction captured here is:
//
//   - primary AIP storage: the source AIP exists in Storage Service
//   - final storage completion: the deployment's configured storage obligations
//     are satisfied
//
// Those are not always the same moment. For local storage they may collapse to
// one timestamp. For replicated storage, final completion may occur later than
// the primary AIP stored date.
//
// ClassifyStorage is the core entry point. It consumes:
//
//   - the pipeline recovery policy, especially required replica locations
//   - normalized Storage Service package metadata
//
// and returns Enduro-specific recovery results such as:
//
//   - not found
//   - local complete
//   - replicated partial
//   - replicated complete
//   - indeterminate
//
// The workflow layer can then use that result to decide whether it should mark
// the collection done, keep it unresolved, or fall back to a full reprocess.
package reconciliation
