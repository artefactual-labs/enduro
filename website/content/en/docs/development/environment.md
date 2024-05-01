---
title: "Environment"
linkTitle: "Environment"
weight: 1
description: >
  Learn to build the development environment suggested by Enduro.
---

Enduro does not have a development mode with all dependencies mocked or stubbed.
The dependencies need to be installed for Enduro to work. To make this process
easier we offer an environment based on Docker Compose where we run all the
dependencies as Docker containers.

But for Enduro itself, we suggest to build it and run it locally because that
makes thing much simpler during development.

## Development dependencies

There are some dependencies that need to be installed:

- [Go][go],
- [Docker Engine][docker-engine] (includes Docker Compose),
- [Node.js][nodejs], and
- [GNU Make][make]

This guide may not work for you if you manage Docker with `sudo`, see
[issue #118][issue-118] for more. It is possible to manage Docker as a non-root
user with some [extra configuration steps][docker-non-root].

## First steps

Spin up the environment with the following command:

    docker compose up --detach

Build the web-based user interface:

    make ui

Finall, build and run Enduro with:

    make run

With Enduro running in the background, you should be able to access the web
interface via <http://127.0.0.1:9000/> or connect to the API, e.g.:

    curl -v 127.0.0.1:9000/collection

## Set up MinIO for object storage

MinIO is one of the services installed automatically with Docker Compose. You
should be able to access the web file browser via <http://127.0.0.1:7460> using
the following credentials:

- Access key: `minio`
- Secret key: `minio123`

Alternatively, you can [install][mc] the MinIO command-line client (mc) and
register the local instance with:

    $ mc config host add enduro http://127.0.0.1:7460 minio minio123
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

## Start a transfer

Uploading a file into the `sips` bucket should trigger the workflow. From the
command-line interface, you can run the following:

    curl -s https://news.ycombinator.com/y18.gif | mc pipe enduro/sips/y18.gif

The workflow should be triggered automatically. It is okay to start Enduro later
since the event is buffered by Redis.

    make run

Additionally, you can visualize workflows and activities from Temporal UI. Try
opening the following link: <http://127.0.0.1:7440/namespaces/default/workflows>.

## Development workflow

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

## Enable tracing

Enable the observability services by editing the root `.env` file as follows:

    COMPOSE_FILE=docker-compose.yml:docker-compose.observability.yml

Start all services again:

    docker compose up -d

Three new services should be ready:

* Grafana (listens on `127.0.0.1:3000`),
* Grafana Agent (listens on `127.0.0.1:12345`), and
* Grafana Tempo (listens on `127.0.0.1:4317`).

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

* Observe that `grafana-agent` logs received traces:

      docker compose logs -f grafana-agent

* Observe that `grafana-tempo` logs stored traces:

      docker compose logs -f grafana-tempo

* Use the [Grafana explorer](http://127.0.0.1:3000/explore) to search and
  visualize traces.


[docker-engine]: https://docs.docker.com/engine/install/
[mc]: https://docs.min.io/docs/minio-client-quickstart-guide.html
[go]: https://golang.org/doc/install
[gopls]: https://github.com/golang/tools/blob/master/gopls/README.md
[node]: https://nodejs.org
[make]: https://www.gnu.org/software/make/
[issue-118]: https://github.com/artefactual-labs/enduro/issues/118
[docker-non-root]: https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user
