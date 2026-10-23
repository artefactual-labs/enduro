# Contributing

## Environment

Enduro does not have a development mode with all dependencies mocked or stubbed.
The dependencies need to be installed for Enduro to work. To make this process
easier we offer an environment based on Docker Compose where we run all the
dependencies as Docker containers.

But for Enduro itself, we suggest to build it and run it locally because that
makes thing much simpler during development.

### Development dependencies

There are some dependencies that need to be installed:

- [Go],
- [Docker Engine][docker-engine] (includes Docker Compose),
- [Node.js][nodejs], and
- [GNU Make][make]

This guide may not work for you if you manage Docker with `sudo`, see
[issue #118][issue-118] for more. It is possible to manage Docker as a non-root
user with some [extra configuration steps][docker-non-root].

### First steps

Spin up the environment with the following command:

```
docker compose up --detach
```

Build the web-based user interface:

```
make frontend
```

Finall, build and run Enduro with:

```
make run
```

With Enduro running in the background, you should be able to access the web
interface via <http://127.0.0.1:9000/> or connect to the API, e.g.:

```
curl -v 127.0.0.1:9000/collection
```

### Set up SeaweedFS for object storage

SeaweedFS is one of the services installed automatically with Docker Compose.
The local S3 endpoint is available at <http://127.0.0.1:7460> using the
following credentials:

- Access key: `minio`
- Secret key: `minio123`

The SeaweedFS filer UI is available at <http://127.0.0.1:7461>.

You can [install][mc] the MinIO command-line client (mc) and register the local
S3 endpoint with:

```
$ mc alias set enduro http://127.0.0.1:7460 minio minio123
Added `enduro` successfully.
```

We provide default configuration so SeaweedFS publishes filer create events to
Enduro's object event webhook. Enduro normalizes those events and publishes them
to our local Redis instance. The `[[watcher.s3]]` entry in `enduro.toml`
consumes those events with `eventSource = "redis"` and `eventFormat = "enduro"`.

The webhook server is configured in `enduro.toml`:

```
[objectEventWebhook]
enabled = true
listen = "0.0.0.0:7480"
redisAddress = "redis://127.0.0.1:7470"
redisList = "object-events"
bucketsPath = "/buckets"
```

SeaweedFS loads its webhook notification configuration from
`hack/seaweedfs/notification.toml`.

List the bucket with:

```
$ mc ls enduro/sips
```

### Start a transfer

Uploading a zipped transfer into the `sips` bucket should trigger the workflow.
From the command-line interface, you can run the following:

```
mc cp ~/Documents/transfer.zip enduro/sips
```

The workflow should be triggered automatically. It is okay to start Enduro later
since the event is buffered by Redis.

```
make run
```

Additionally, you can visualize workflows and activities from Temporal UI. Try
opening the following link:
<http://127.0.0.1:7440/namespaces/default/workflows>.

### Development workflow

You can use the standard Go build workflows such `go build` or `go install` but
we provide a couple of shortcuts that use our custom build directory and flags:

```
# Build the binary in ./build/enduro
make enduro-dev

# Build and run
make run
```

If you are looking for a source-code editor, Visual Studio Code has great Go
support. It uses the official language server, [gopls], which is it not stable
yet - it is recommendable to read their docs and keep it up to date.

Other useful commands are `make lint` and `make test`. This project uses
GolangCI-Lint, an aggregator of many linters. It is installed with `make tools`.
You can enable it in Visual Studio Code as follows:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"]
}
```

### Dagger smoke harness

Enduro also has an experimental Dagger-based smoke harness under `hack/dagger`.
The same harness runs in the dedicated `smoke-tests` job in the `Test` GitHub
Actions workflow. It can also be run manually before pushing changes that affect
transfer publishing or Archivematica integration.

Install a recent version of Dagger and make sure Docker is available. From the
repository root, run:

```
make test-smoke
```

Because this harness starts real services and Archivematica workers, it is
slower and heavier than `make test`. Treat it as an end-to-end smoke check for
the transfer publishing workaround rather than a replacement for unit or
workflow tests. The harness also exports Go coverage from the Enduro process; CI
uploads that profile to Codecov under the `go-smoke` flag.

#### MinIO compatibility tests (local only)

CI runs the object storage smoke tests against SeaweedFS and argmin. MinIO is
excluded because its [community server is no longer maintained][minio-status]
and the legacy releases have known security vulnerabilities. Compatibility with
MinIO is checked manually when investigating reports or changing its
integration; a passing CI run does not verify MinIO compatibility.

The local scenario uploads a transfer through S3, exercises MinIO's native Redis
notifications and Enduro's S3 watcher, and verifies the resulting Archivematica
AIP. It starts its own services with dummy credentials. Use disposable test data
and keep these services separate from production systems.

With Docker running and Dagger installed, run this command from the repository
root:

```sh
dagger -m hack/dagger call object-storage-smoke-test \
  --source . \
  --provider minio-latest \
  export --path hack/dagger/runtime/object-storage-artifacts-minio-latest
```

On success, the output directory contains the downloaded AIP, Temporal workflow
history, and coverage artifacts. The test starts Archivematica and can take
several minutes. `make test-smoke-object-storage` runs all three providers
locally, including MinIO; it is broader than the CI matrix.

The image is selected by `minioLatestImage` in `hack/dagger/main.go`. To
investigate a compatibility report, set that constant locally to the reported
image tag or digest and rerun the command. Record the image reference, Enduro
revision, and test output with the report. A result applies to the tested
version and scenario, not every MinIO deployment.

Legacy image availability is a separate limitation: on October 23, 2026, the
pinned Quay image and the Quay and Docker Hub `latest` tags returned
authorization errors. If image resolution fails, obtain an accessible image from
your existing deployment or a trusted registry and update `minioLatestImage`
before retrying. An image-pull failure does not establish an Enduro
compatibility failure.

### Enable tracing

Enable the observability services by editing the root `.env` file as follows:

```
COMPOSE_FILE=docker-compose.yml:docker-compose.observability.yml
```

Start all services again, removing containers for services that are no longer
defined in this Compose project:

```
docker compose up -d --remove-orphans
```

Three new services should be ready:

- Grafana (listens on `127.0.0.1:3000`),
- Grafana Alloy (accepts OTLP/gRPC on `127.0.0.1:12345`), and
- Grafana Tempo (listens on `127.0.0.1:4317`).

Alloy replaces the retired Grafana Agent and keeps the same OTLP/gRPC endpoint.
The `--remove-orphans` flag removes the old `grafana-agent` container, if
present, so that it releases port `12345` before Alloy starts.

Tempo runs in monolithic mode and stores traces in the `tempo-data` named
volume. Docker initializes this volume with the ownership required by Tempo's
container user. Existing traces in `hack/stuff/tempo-data` are left untouched
and are not imported into the new volume. The Alloy debug exporter logs received
spans and requires the experimental stability flag set in Compose.

Enable tracing in `enduro.toml`:

```toml
[telemetry.traces]
enabled = true
address = "127.0.0.1:12345"
ratio = 1.0
```

Run enduro:

```
make run
```

Things that you can do:

- Observe that `grafana-alloy` logs received traces:

  ```
  docker compose logs -f grafana-alloy
  ```

- Observe that `grafana-tempo` logs stored traces:

  ```
  docker compose logs -f grafana-tempo
  ```

- Use the [Grafana explorer] to search and visualize traces.

## API changes

Enduro uses the [Goa framework][goa] to build the API. Goa provides a
[design language][goa-dsl] which is our source of truth from which both behavior
and docs are derived.

Goa generates all the code that relates to the communication transport as well
as the API specification using OpenAPI. Enduro provides the implementation of
the services where we include the business logic. This idea is described by
Robert C. Martin in [The Clean Architecture][clean-arch].

See also:

- [OpenAPI 3.2 schema][openapi3-2-json]
- [OpenAPI 3.0 schema][openapi3-json]

### API design

Our API definition can be found in the [api/design][design-pkg] package. The
following example shows how we've described the `DELETE /collection` API.

```go
Method("delete", func() {
    Description("Delete collection by ID")
    Payload(func() {
        Attribute("id", UInt, "Identifier of collection to delete")
        Required("id")
    })
    Error("not_found", NotFound, "Collection not found")
    HTTP(func() {
        DELETE("/{id}")
        Response(StatusNoContent)
        Response("not_found", StatusNotFound)
    })
})
```

With this design language we describe our API services, methods and types. It is
also possible to implement some advanced features such different authentication
methods or streaming of contents.

### Backend development

After making new changes to the API design, the developer should run:
`make gen-goa` which generates all the code under `internal/api/gen`, including
the OpenAPI description of the API for the HTTP transport:
[`openapi3.2.json`][openapi3-2-json] and [`openapi3.json`][openapi3-json]. The
3.0 document remains the input for the generated UI clients because it is
supported by more client-generation tools.

In the example above, we added a `Delete` method to the `Collection` service.
The corresponding Go interface gets a new method:

```go
Delete(context.Context, *DeletePayload) (err error)
```

It's now up to the developer to implement the expected functionality.

### Frontend development

We use `openapi-generator-cli` to generate the client code after the OpenAPI
description of the API. Run `make frontend-client` to generate all the
TypeScript code under `frontend/app/openapi-generator` which is used by the
Enduro frontend.

You should see new models and methods added, like the new `collectionDelete`
method:

```ts
async collectionDelete(requestParameters: CollectionDeleteRequest): Promise<void> {
    await this.collectionDeleteRaw(requestParameters);
}
```

## Software dependencies

### Go

The version of Go used by this project is specified in the `go.mod` file.

### Go modules

Use `make deps` to list available updates.

When updating Goa, keep its version in `.bine.toml` aligned with `go.mod`, then
run `make gen-goa` and `make frontend-client` to regenerate the API and client.

### OpenAPI Generator

In Makefile (target `frontend-client`), as a Docker container.

### UI

Use `npm run upgrade:deps --prefix frontend` to select compatible dependency
updates, then `npm install --prefix frontend` to install them and update the
lockfile.

### Other tools

See `.bine.toml` for pinned build and development tools. Dagger's engine and Go
dependencies are configured in `hack/dagger/dagger.json` and
`hack/dagger/go.mod`. Keep the Playwright version in
`hack/dagger/smoke/package.json` aligned with its container image in
`hack/dagger/main.go`.

The MinIO smoke-test server uses the published Quay image pinned in
`hack/dagger/main.go`. The `minio-latest` provider name is a legacy label; it
does not track a moving `latest` tag. This image predates the October 2025
security fix and is retained only for legacy compatibility testing.

## Releases

The release process is automated via GitHub Actions. See
[`release.yml`][release-workflow] for more - we plan to develop it further.

The artifacts associated with a release are automatically published after the
git tag is submitted to the repository. We use annotated tags. For example:

```
git tag -a "v0.20.0" -m "Enduro v0.20.0"
git push --follow-tags
```

The `release` workflow is started as soon as the tag makes it to GitHub. The
artifacts will start building right away and they will appear in the
[Releases page][release-github-page] as soon as they're ready.

The workflow runs are recorded by GitHub for auditing purposes, e.g. check out
the [workflow run][first-release-run] that made our very first release.

[clean-arch]: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
[design-pkg]: https://github.com/artefactual-labs/enduro/tree/main/internal/api/design
[docker-engine]: https://docs.docker.com/engine/install/
[docker-non-root]: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user
[first-release-run]: https://github.com/artefactual-labs/enduro/commit/0f17f7fa3beb88f7039cf9b6e1dc1a6f4af66f51/checks
[go]: https://golang.org/doc/install
[goa]: https://goa.design/
[goa-dsl]: https://godoc.org/goa.design/goa/dsl
[gopls]: https://github.com/golang/tools/blob/master/gopls/README.md
[grafana explorer]: http://127.0.0.1:3000/explore
[issue-118]: https://github.com/artefactual-labs/enduro/issues/118
[make]: https://www.gnu.org/software/make/
[mc]: https://docs.min.io/docs/minio-client-quickstart-guide.html
[minio-status]: https://github.com/minio/minio
[nodejs]: https://nodejs.org
[openapi3-2-json]: https://github.com/artefactual-labs/enduro/blob/main/internal/api/gen/http/openapi3.2.json
[openapi3-json]: https://github.com/artefactual-labs/enduro/blob/main/internal/api/gen/http/openapi3.json
[release-github-page]: https://github.com/artefactual-labs/enduro/releases
[release-workflow]: https://github.com/artefactual-labs/enduro/blob/main/.github/workflows/release.yml
