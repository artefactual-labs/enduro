# Recovery Guide

This guide covers Enduro's optional recovery flow for collections where
Archivematica has already created the package identity and Enduro can use
Storage Service to verify whether that package is really present and complete.
This is especially useful when ingest fails during or after Storage Service
replication and a normal retry would risk creating a duplicate AIP and duplicate
replicas.

In Enduro, "recovery" means that retry does not immediately start a new transfer
once Enduro already knows which package it is dealing with. Instead, retry first
checks Storage Service to see whether that known package is already present and
complete, partially stored, or missing.

So the goal of retry changes:

- first reconcile the known package in Storage Service
- finish from that existing package if Storage Service confirms it is complete
- start a fresh transfer only if Storage Service shows that package is missing

This is mainly meant to solve the case where Archivematica ingest fails during
or after replica creation, Enduro records the collection as `error`, but the
source AIP may already exist in Storage Service. Without recovery, retry can
create a duplicate AIP and duplicate replicas. With recovery enabled, Enduro
reconciles Storage Service first and only falls back to a fresh transfer when
reconciliation shows that the AIP is not present.

The important distinction is:

- once Enduro has recorded the Archivematica package UUID, it knows which
  package to reconcile
- it does not, by itself, mean Storage Service has confirmed that a stored and
  complete AIP exists for that UUID
- reconciliation is the step that checks Storage Service and decides whether
  that known UUID actually satisfies the configured storage completion rule

This recovery flow is about reconciliation and safer retry behavior. It does not
repair replicas or trigger replica creation by itself yet.

## How retry changes

When recovery is disabled for a pipeline, Enduro keeps the current behavior:
retry starts a fresh reprocess.

When recovery is enabled for a pipeline:

- if Enduro does not yet know the package UUID, retry still follows the normal
  full reprocess path
- if Enduro already knows the package UUID, retry enters a reconciliation branch
  before Enduro creates any new transfer
- if Storage Service confirms storage is complete, Enduro finishes the
  collection without creating a duplicate transfer
- if Storage Service returns `not_found`, Enduro clears the old identifiers and
  continues into the normal full reprocess path
- if Storage Service says the package exists but storage is incomplete, or the
  state is indeterminate, Enduro stops instead of creating a duplicate AIP

Recovery-enabled pipelines also run reconciliation after ingest. That means the
final `completed_at` can come from Storage Service confirmation, not only from
Archivematica polling. It also means an ingest error can still end as `done`
when Storage Service confirms that the package already satisfies the configured
storage rule. If Storage Service is briefly behind and returns `not_found`, or
if package state is still not clear enough for Enduro to confirm completion,
Enduro retries that post-ingest reconciliation for up to 5 minutes before
failing. If Storage Service reports that required locations are still missing,
Enduro fails immediately instead of waiting for replica repair to happen on its
own.

## Current limitations and operator impact

The current implementation keeps the pipeline slot until the whole workflow
returns from the ingest path, including any post-ingest reconciliation retries.
In other words, Archivematica may already be finished with the transfer, but
Enduro still counts that collection against the pipeline `capacity` while it is
waiting for Storage Service to confirm the outcome.

This matters most when recovery is enabled and Storage Service is temporarily
behind Archivematica. During that period:

- Enduro retries post-ingest reconciliation every 15 seconds for up to 5
  minutes, but only for temporary `not_found` or indeterminate results
- each affected collection continues to occupy one pipeline slot for that entire
  retry window
- unrelated transfers targeting the same pipeline can remain queued even though
  Archivematica ingest work for the earlier collection has already ended

Operationally, this means recovery can reduce effective throughput on busy
pipelines when Storage Service is slow to reflect recently created packages. The
impact is most visible on pipelines with low `capacity`, where a single delayed
reconciliation can block a large share of the available work.

This is a known limitation of the current workflow design. A future improvement
could separate post-ingest reconciliation from Archivematica slot accounting so
Enduro can release the pipeline slot earlier and continue the Storage Service
checks under different concurrency rules. That is not implemented today and
would require additional workflow and retry-state changes to keep recovery
behavior safe and predictable.

## When to use recovery

Use recovery when you want Enduro to avoid duplicate reprocessing in cases where
Archivematica has already created the package UUID, but Storage Service is the
real source of truth for whether that package is present and complete. This is
most useful when storage completion may depend on replica locations, or when
Archivematica and Storage Service can be briefly out of sync.

Recovery is disabled by default. Configuring `storageServiceURL` alone does not
change retry behavior. Enduro only uses the recovery flow when
`reconcileExistingAIP = true` is set for that pipeline.

## Recovery configuration

Recovery is configured per pipeline.

The `[pipeline.recovery]` block applies to the most recent `[[pipeline]]` entry
above it. If you define two pipelines, each pipeline must have its own recovery
block.

Minimal example:

```toml
[[pipeline]]
name = "am"
storageServiceURL = "http://test:test@127.0.0.1:62081"

[pipeline.recovery]
reconcileExistingAIP = true

# `requiredLocations` is omitted here, so Enduro uses the primary-only rule.
```

Replica-aware example:

```toml
[[pipeline]]
name = "am"
storageServiceURL = "http://test:test@127.0.0.1:62081"

[pipeline.recovery]
reconcileExistingAIP = true
requiredLocations = ["primary-location-id", "replica-location-id"]
```

Use the minimal form when the primary package alone is enough to satisfy your
storage rule. Omitting `requiredLocations` has the same effect as setting it to
`[]`. Add `requiredLocations` only when specific Storage Service locations must
also be confirmed before Enduro should treat the collection as fully stored.

Example based on a fuller pipeline definition:

```toml
[[pipeline]]
name = "am"
baseURL = "http://127.0.0.1:62080"
user = "test"
key = "test"
transferDir = "~/.am/ss-location-data"
processingDir = ""
processingConfig = "automated"
# transferLocationID = "88f6b517-c0cc-411b-8abf-79544ce96f54"
storageServiceURL = "http://test:test@127.0.0.1:62081"
capacity = 3
retryDeadline = "10m"
transferDeadline = "24h"
unbag = false

[pipeline.recovery]
# Enable reconciliation for retries that already have an AIP.
reconcileExistingAIP = true

# Optional. If omitted or set to an empty list, Enduro treats the primary
# package alone as the completion condition.
#
# Set this only when Enduro should require specific Storage Service locations
# before treating storage as complete.
requiredLocations = [
  "primary-location-id",
  "replica-location-id",
]
```

### Two-pipeline example

When different pipelines need different recovery policies, repeat
`[pipeline.recovery]` under each `[[pipeline]]` entry:

```toml
[[pipeline]]
name = "am1"
baseURL = "http://127.0.0.1:62080"
user = "test"
key = "test"
transferDir = "~/.am1/transfers"
processingDir = ""
processingConfig = "automated"
storageServiceURL = "http://test:test@127.0.0.1:62081"
capacity = 3
retryDeadline = "10m"
transferDeadline = "24h"
unbag = false

[pipeline.recovery]
reconcileExistingAIP = true
requiredLocations = [
  "am1-primary-location-id",
  "am1-replica-location-id",
]

[[pipeline]]
name = "am2"
baseURL = "http://127.0.0.1:62090"
user = "test"
key = "test"
transferDir = "~/.am2/transfers"
processingDir = ""
processingConfig = "automated"
storageServiceURL = "http://test:test@127.0.0.1:62091"
capacity = 2
retryDeadline = "15m"
transferDeadline = "24h"
unbag = false

[pipeline.recovery]
reconcileExistingAIP = true

# Empty means the primary package alone is the completion condition.
requiredLocations = []
```

## Recovery settings

### `reconcileExistingAIP` (Boolean)

When enabled, retries for collections whose package UUID is already known to
Enduro will check Storage Service before creating a new transfer.

E.g.: `false`

### `requiredLocations` (Array(String))

Storage Service locations that must be confirmed before storage is considered
complete. This list may include the primary package location and any replica
locations. If omitted or left empty, Enduro treats the primary package alone as
the completion condition. If the list is populated, Enduro requires every listed
location before the collection is considered complete.

E.g.: `["primary-location-id", "replica-location-id"]`

Guidance:

- omit this setting, or set it to `[]`, when your completion rule is "the
  primary package alone is enough"
- include the primary package location when that location is part of your
  explicit completion policy
- include replica locations only when those replicas must exist before the
  collection should be considered fully stored
- only include locations that operators are prepared to monitor; every required
  location becomes part of the success condition

## Finding valid Storage Service location IDs

`requiredLocations` must use Storage Service location UUIDs. Enduro does not
invent these values.

Operators should obtain them from Archivematica Storage Service and record the
UUIDs they intend to use in pipeline configuration. In practice, this means
looking up the primary package location and any replica locations in Storage
Service before enabling recovery.

If multiple pipelines have different storage policies, each pipeline can use a
different `requiredLocations` list.

## Reconciliation after ingest

When recovery is enabled and Enduro already knows the package UUID, Enduro also
runs reconciliation after ingest processing.

Operationally, this means:

- if Storage Service confirms completion, Enduro can use Storage Service timing
  as the final completion time
- if Archivematica ingest reports an error, Enduro can still finish as `done`
  when Storage Service confirms that the AIP already satisfies the configured
  storage rule
- if Storage Service temporarily returns `not_found`, or returns an
  indeterminate result while package state is still settling, Enduro retries
  reconciliation every 15 seconds for up to 5 minutes before treating that
  result as final
- this short retry window applies whether the Archivematica ingest step ended
  successfully or with an error, because Storage Service can lag behind the
  ingest-side outcome
- during that retry window, the collection still occupies a pipeline slot in
  Enduro's current implementation
- if Storage Service reports that required locations are still missing, the
  workflow fails immediately rather than guessing that storage is complete
- if Storage Service still reports `not_found` or indeterminate after that
  window, the workflow fails rather than guessing that storage is complete

## When Enduro refuses to start a new transfer

When recovery is enabled and Enduro already knows the package UUID, it will not
start a new transfer in these cases:

- Storage Service reports that the package is present and storage is complete
- Storage Service reports that the package is present, but one or more required
  locations are still missing
- Storage Service responds, but Enduro cannot determine storage completion
  safely from the returned state

In the first case, Enduro completes the collection successfully. In the latter
two cases, Enduro leaves the collection in an error state rather than risk
creating a duplicate AIP.

## Reconciliation statuses for operators

Collections that have storage reconciliation data expose it in the collection
detail view under the `Storage` tab.

The status values mean:

- `pending`: Enduro has not recorded a final storage conclusion yet
- `partial`: the primary AIP exists, but required storage is still incomplete
- `complete`: the AIP satisfies the configured storage completion rule
- `unknown`: Enduro could not determine storage completion safely

In practical terms:

- `complete` means the collection can be treated as stored
- `partial` means operators should expect at least one required replica or
  required location to still be missing
- `unknown` means operators should inspect Storage Service and the recorded
  reconciliation error before retrying again
- `pending` usually means reconciliation has not happened yet, or no final state
  has been recorded for that collection

## Storage tab in collection detail

The `Storage` tab is the operator view for this feature. It shows:

- a short storage summary
- reconciliation status
- primary AIP stored time
- last reconciliation check time
- reconciliation error, when present

If Enduro does not yet know the package UUID, the tab explains that storage
reconciliation is not available yet. If Enduro knows the package UUID but no
reconciliation data has been recorded yet, the tab shows that no storage
reconciliation result is available yet. In other words, Enduro may know the
Archivematica package UUID before it has confirmed the storage outcome for that
UUID.
