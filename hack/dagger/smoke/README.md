# Enduro Smoke Tests

These Playwright tests exercise Enduro against an ambox Archivematica
environment started by Dagger.

Dagger owns service orchestration, shared runtime directories, and artifact
export. It also builds Enduro with Go binary coverage instrumentation, stops
the service after the scenarios complete, and exports a coverage profile with
the rest of the artifacts. The tests own the user-visible scenarios:

- submit a zip through the filesystem watcher,
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

- `minio-legacy`: fixture-backed MinIO native Redis notifications
  (`eventFormat = "minio"`),
- `minio-latest`: env-configured MinIO native Redis notifications
  (`eventFormat = "minio"`),
- `seaweedfs`: SeaweedFS filer webhooks through Enduro's object event webhook
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
