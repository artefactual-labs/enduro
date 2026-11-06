# Enduro Smoke Tests

These Playwright tests exercise Enduro against an ambox Archivematica
environment started by Dagger. Use Dagger 1.0.0-beta.24 to match the module
configuration in `hack/dagger/dagger-module.toml`.

The repository workspace is configured in `dagger.toml`. After changing the
module API, run `dagger generate -y` and commit the generated bindings and
`dagger.lock`. The Go SDK currently resets the OpenTelemetry logging
replacements to v0.16.0 during generation; retain the v0.19.0 replacements
documented in `hack/dagger/go.mod` before committing. Refresh locked
dependencies deliberately with `dagger lock update`.

Dagger owns service orchestration, shared runtime directories, and artifact
export. It can import a production image archive, pull an immutable registry
reference, or build Enduro from source with Go binary coverage instrumentation.
The tests own the user-visible scenarios:

- submit a zip through the filesystem watcher,
- recover a failed production-system receipt with an operator decision,
- submit a directory through the Nuxt batch-import form,
- wait for Enduro collections to complete,
- download each generated AIP and inspect its METS file,
- retrieve the completed filesystem transfer's trace from LGTM and check its
  processing-workflow, publish-transfer, and transfer-activity spans,
- inspect Temporal history for the expected workflow activities.

The main smoke suite starts a disposable Grafana LGTM container and enables
Enduro's OTLP/gRPC tracing with full sampling. It waits for LGTM's Tempo
datasource before submitting the first transfer, then queries that transfer's
workflow ID through Grafana. Missing traces or expected spans fail the suite.
The search response, complete trace, and workflow/trace IDs are exported as
`tempo-search.json`, `tempo-trace.json`, and `telemetry-report.txt` alongside
the other CI artifacts. No telemetry data is persisted between runs.

Run the suite from the repository root:

```sh
make test-smoke
```

The Make target runs:

```sh
dagger -m hack/dagger api call smoke-tests --source . export --path hack/dagger/runtime/artifacts
```

## Runtime image modes

Without an image argument, the harness builds an instrumented Enduro binary from
source and packages it with the production `Dockerfile`, static base, nonroot
user, entrypoint, and default `/etc/enduro/enduro.toml` configuration path. It
stops Enduro after the scenarios and exports a coverage profile alongside the
other artifacts. `--frontend-assets frontend/.output/public` can supply
previously built public assets to this source coverage mode; otherwise Dagger
builds the frontend.

To test production images, follow the [container guide] to build and export the
snapshot once, then select the archive for the native platform:

```sh
dagger -m hack/dagger api call smoke-tests --source . \
  --image-archive dist/containers/enduro-amd64.tar \
  export --path hack/dagger/runtime/artifacts
```

Alternatively, pass an immutable registry digest:

```sh
dagger -m hack/dagger api call smoke-tests --source . \
  --image-ref ghcr.io/artefactual-labs/enduro@sha256:REPLACE_WITH_INDEX_DIGEST \
  export --path hack/dagger/runtime/artifacts
```

The same `--image-archive` and `--image-ref` options apply to `startup-test`,
`object-storage-smoke-test`, and `object-storage-smoke-tests`. The two image
options are mutually exclusive. They use the existing production binary without
rebuilding or collecting coverage, and cannot be combined with
`--frontend-assets`. Dagger mounts the scenario configuration at the image's
default configuration path and runs its normal entrypoint.

For a shorter native startup check without Archivematica:

```sh
make container-startup
```

This imports the native archive from `dist/containers` and checks database
migrations, the API, embedded dashboard assets, and shutdown with MySQL and
Temporal. CI runs startup acceptance on native AMD64 and ARM64. Full acceptance
also runs the main Archivematica, SeaweedFS, and argmin image suites on AMD64.
Snapshot jobs receive archives without registry credentials; release jobs test
the candidate by immutable digest. A separate source-built smoke suite collects
coverage. CI reuses one frontend build for the snapshot images and the coverage
build.

### Branch iteration

Pushes to `dev/containers` run both native startup checks and a smaller source
coverage check without the full service suites. Run the coverage check locally
with:

```sh
dagger -m hack/dagger api call coverage-check --source . \
  export --path hack/dagger/runtime/coverage-check-artifacts
```

Add `--frontend-assets frontend/.output/public` to reuse existing public assets;
omit it to let Dagger build them. The check builds and runs the real
instrumented Enduro binary with `--version`, then exports Go coverage data. It
starts no services and verifies the coverage build and export path, not workflow
behavior. A cold compile can still take several minutes.

Pull requests and `main` run the full image suites and source smoke coverage;
releases run full image acceptance. For a final full run on `dev/containers`,
manually dispatch the Test workflow with `full-tests` enabled (the default).

## Object storage

The object-storage smoke suite exercises S3 watcher ingestion through the
supported local object storage paths:

- `minio-latest`: env-configured MinIO native Redis notifications
  (`eventFormat = "minio"`),
- `seaweedfs`: SeaweedFS filer webhooks through Enduro's object event webhook
  (`eventFormat = "enduro"`),
- `argmin`: argmin S3 storage with a test-injected normalized Redis event
  (`eventFormat = "enduro"`).

These object-storage scenarios do not start LGTM and keep tracing disabled.

Run the object-storage suite from the repository root:

```sh
make test-smoke-object-storage
```

The Make target runs:

```sh
dagger -m hack/dagger api call object-storage-smoke-tests --source . export --path hack/dagger/runtime/object-storage-artifacts
```

The object-storage tests build and use `hack/s3put`, a small S3-compatible
upload helper, so the same upload path can be used against each local provider
without depending on a provider-specific CLI.

Run only argmin with:

```sh
dagger -m hack/dagger api call object-storage-smoke-test --source . --provider argmin export --path hack/dagger/runtime/object-storage-artifacts-argmin
```

GitHub Actions includes argmin in the storage matrix. Dagger builds a pinned
[upstream] revision with Cargo caches, without a Dockerfile.

Implementors can experiment with argmin as S3 storage, but the tested revision
has no native notifications: the producer must deliver an Enduro
`object.created` event to the configured Redis list after a successful upload.
This test supplies that event and verifies ingestion through AIP creation.

Uploading and notifying are separate operations, not one transaction. Producers
need durable tracking, retries, and reconciliation with Enduro to recover missed
ingestion. Retries can duplicate events, so use stable, unique object keys and
plan for duplicate handling. The Redis watcher removes events before processing;
neither this delivery path nor the smoke test guarantees exactly-once processing
or crash recovery.

[container guide]: ../../../docs/containers.md
[upstream]: https://github.com/justincormack/argmin
