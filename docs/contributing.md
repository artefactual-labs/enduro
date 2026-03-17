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

- [Go][go],
- [Docker Engine][docker-engine] (includes Docker Compose),
- [Node.js][nodejs], and
- [GNU Make][make]

This guide may not work for you if you manage Docker with `sudo`, see
[issue #118][issue-118] for more. It is possible to manage Docker as a non-root
user with some [extra configuration steps][docker-non-root].

### First steps

Spin up the environment with the following command:

    docker compose up --detach

Build the web-based user interface:

    make ui

Finall, build and run Enduro with:

    make run

With Enduro running in the background, you should be able to access the web
interface via <http://127.0.0.1:9000/> or connect to the API, e.g.:

    curl -v 127.0.0.1:9000/collection

### Set up MinIO for object storage

MinIO is one of the services installed automatically with Docker Compose. You
should be able to access the web file browser via <http://127.0.0.1:7460> using
the following credentials:

- Access key: `minio`
- Secret key: `minio123`

Alternatively, you can [install][mc] the MinIO command-line client (mc) and
register the local instance with:

    $ mc alias set enduro http://127.0.0.1:7460 minio minio123
    Added `enduro` successfully.

We provide some default configuration so MinIO publishes events via our local
Redis instance. Validate the configuration with:

    $ mc admin config get enduro notify_redis
    notify_redis:1 format=access address=redis:6379 password= key=minio-events queue_dir=/tmp/events queue_limit=10000
    notify_redis enable=off format=namespace address= key= password= queue_dir= queue_limit=0

    $ mc event list enduro/sips
    arn:minio:sqs::1:redis   s3:ObjectCreated:*   Filter:

List the bucket with:

    $ mc ls enduro/sips
    [2020-04-29 13:28:32 PDT]  4.6KiB archivematica.png

### Start a transfer

Uploading a zipped transfer into the `sips` bucket should trigger the
workflow. From the command-line interface, you can run the following:

    mc cp ~/Documents/transfer.zip enduro/sips

The workflow should be triggered automatically. It is okay to start Enduro later
since the event is buffered by Redis.

    make run

Additionally, you can visualize workflows and activities from Temporal UI. Try
opening the following link: <http://127.0.0.1:7440/namespaces/default/workflows>.

### Development workflow

You can use the standard Go build workflows such `go build` or `go install` but
we provide a couple of shortcuts that use our custom build directory and flags:

    # Build the binary in ./build/enduro
    make enduro-dev

    # Build and run
    make run

If you are looking for a source-code editor, Visual Studio Code has great Go
support. It uses the official language server, [gopls][gopls], which is it not
stable yet - it is recommendable to read their docs and keep it up to date.

Other useful commands are `make lint` and `make test`. This project uses
GolangCI-Lint, an aggregator of many linters. It is installed with `make tools`.
You can enable it in Visual Studio Code as follows:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"]
}
```

### Enable tracing

Enable the observability services by editing the root `.env` file as follows:

    COMPOSE_FILE=docker-compose.yml:docker-compose.observability.yml

Start all services again:

    docker compose up -d

Three new services should be ready:

- Grafana (listens on `127.0.0.1:3000`),
- Grafana Agent (listens on `127.0.0.1:12345`), and
- Grafana Tempo (listens on `127.0.0.1:4317`).

Enable tracing in `enduro.toml`:

```toml
[telemetry.traces]
enabled = true
address = "127.0.0.1:12345"
ratio = 1.0
```

Run enduro:

    make run

Things that you can do:

- Observe that `grafana-agent` logs received traces:

      docker compose logs -f grafana-agent

- Observe that `grafana-tempo` logs stored traces:

      docker compose logs -f grafana-tempo

- Use the [Grafana explorer](http://127.0.0.1:3000/explore) to search and
  visualize traces.

## API changes

Enduro uses the [Goa framework][goa] to build the API. Goa provides a
[design language][goa-dsl] which is our source of truth from which both
behavior and docs are derived.

Goa generates all the code that relates to the communication transport as well
as the API specification using OpenAPI. Enduro provides the implementation of
the services where we include the business logic. This idea is described by
Robert C. Martin in [The Clean Architecture][clean-arch].

See also:

- [OpenAPI schema (`openapi3.json`)][openapi3-json]

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

With this design language we describe our API services, methods and types. It
is also possible to implement some advanced features such different
authentication methods or streaming of contents.

### Backend development

After making new changes to the API design, the developer should run:
`make gen-goa` which generates all the code under `internal/api/gen`, including
the OpenAPI description of the API for the HTTP transport:
[`openapi3.json`][openapi3-json].

In the example above, we added a `Delete` method to the `Collection` service.
The corresponding Go interface gets a new method:

```go
Delete(context.Context, *DeletePayload) (err error)
```

It's now up to the developer to implement the expected functionality.

### Frontend development

We use `openapi-generator-cli` to generate the client code after the OpenAPI
description of the API. Run `make ui-client` to generate all the TypeScript
code under `ui/src/client` which is used by the Enduro frontend.

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

When updating Goa, make sure `hack/make/dep_goa.mk` is using the same version.

### OpenAPI Generator

In Makefile (target `ui-client`), as a Docker container.

### UI

Use `npm run deps-minor` to install available updates.

### Other tools

See `hack/make/dep_*.mk`.

## Releases

The release process is automated via GitHub Actions. See
[`release.yml`][release-workflow] for more - we plan to develop it further.

The artifacts associated with a release are automatically published after the
git tag is submitted to the repository. We use annotated tags. For example:

    git tag -a "v0.20.0" -m "Enduro v0.20.0"
    git push --follow-tags

The `release` workflow is started as soon as the tag makes it to GitHub. The
artifacts will start building right away and they will appear in the
[Releases page][release-github-page] as soon as they're ready.

The workflow runs are recorded by GitHub for auditing purposes, e.g. check out
the [workflow run][first-release-run] that made our very first release.

[docker-engine]: https://docs.docker.com/engine/install/
[mc]: https://docs.min.io/docs/minio-client-quickstart-guide.html
[go]: https://golang.org/doc/install
[gopls]: https://github.com/golang/tools/blob/master/gopls/README.md
[nodejs]: https://nodejs.org
[make]: https://www.gnu.org/software/make/
[issue-118]: https://github.com/artefactual-labs/enduro/issues/118
[docker-non-root]: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user
[goa]: https://goa.design/
[goa-dsl]: https://godoc.org/goa.design/goa/dsl
[design-pkg]: https://github.com/artefactual-labs/enduro/tree/main/internal/api/design
[clean-arch]: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
[openapi3-json]: https://github.com/artefactual-labs/enduro/blob/main/internal/api/gen/http/openapi3.json
[release-workflow]: https://github.com/artefactual-labs/enduro/blob/main/.github/workflows/release.yml
[release-github-page]: https://github.com/artefactual-labs/enduro/releases
[first-release-run]: https://github.com/artefactual-labs/enduro/commit/0f17f7fa3beb88f7039cf9b6e1dc1a6f4af66f51/checks
