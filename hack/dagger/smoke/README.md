# Enduro Smoke Tests

These Playwright tests exercise Enduro against an ambox Archivematica
environment started by Dagger.

Dagger owns service orchestration, shared runtime directories, and artifact
export. It also builds Enduro with Go binary coverage instrumentation, stops
the service after the scenarios complete, and exports a coverage profile with
the rest of the artifacts. The tests own the user-visible scenarios:

- submit a zip through the filesystem watcher,
- recover a failed production-system receipt with an operator decision,
- submit a directory through the Nuxt batch-import form,
- wait for Enduro collections to complete,
- download each generated AIP and inspect its METS file,
- inspect Temporal history for the expected workflow activities.

Run the suite from the repository root:

```sh
make test-smoke
```

The Make target runs:

```sh
dagger -m hack/dagger call smoke-tests --source . export --path hack/dagger/runtime/artifacts
```

The object-storage smoke suite exercises S3 watcher ingestion through the
supported local object storage paths:

- `minio-latest`: env-configured MinIO native Redis notifications
  (`eventFormat = "minio"`),
- `seaweedfs`: SeaweedFS filer webhooks through Enduro's object event webhook
  (`eventFormat = "enduro"`),
- `argmin`: argmin S3 storage with a test-injected normalized Redis event
  (`eventFormat = "enduro"`).

Run the object-storage suite from the repository root:

```sh
make test-smoke-object-storage
```

The Make target runs:

```sh
dagger -m hack/dagger call object-storage-smoke-tests --source . export --path hack/dagger/runtime/object-storage-artifacts
```

The object-storage tests build and use `hack/s3put`, a small S3-compatible
upload helper, so the same upload path can be used against each local provider
without depending on a provider-specific CLI.

Run only argmin with:

```sh
dagger -m hack/dagger call object-storage-smoke-test --source . --provider argmin export --path hack/dagger/runtime/object-storage-artifacts-argmin
```

GitHub Actions includes argmin in the storage matrix. Dagger builds a pinned
[upstream](https://github.com/justincormack/argmin) revision with Cargo caches,
without a Dockerfile.

Implementors can experiment with argmin as S3 storage, but the tested revision
has no native notifications: the producer must deliver an Enduro
`object.created` event to the configured Redis list after a successful upload.
This test supplies that event and verifies ingestion through AIP creation.

Uploading and notifying are separate operations, not one transaction. Producers
need durable tracking, retries, and reconciliation with Enduro to recover
missed ingestion. Retries can duplicate events, so use stable, unique object
keys and plan for duplicate handling. The Redis watcher removes events before
processing; neither this delivery path nor the smoke test guarantees
exactly-once processing or crash recovery.
