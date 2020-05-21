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

* [Go][go],
* [Docker Compose][docker-compose],
* [Yarn][yarn], and
* [GNU Make][make]

## First steps

Spin up the environment with the following command:

    docker-compose up --detach

Cadence will crash right away because the database has not been set up properly.
Run the following command to introduce the MySQL tables needed by Cadence:

    make cadence-seed

In a minute or less, Cadence should be up again. Since Cadence is a multitenant
service, we need to create a Cadence domain for Enduro:

    make cadence-domain

Now we need to build some Go tools we're going to use during development:

    make tools

To build our first binary succesfully, we need to generate the database
migrations and the web interface with:

    make migrations ui

We're ready to run our first build:

    make

## Set up MinIO for object storage

MinIO is one of the services installed automatically with Docker Compose. You
should be able to access the web file browser via http://127.0.0.1:7460 using
the following credentials:

* Access key: `minio`
* Secret key: `minio123`

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

Additionally, you can visualize workflows and activities from Cadence Web. Try
opening the following link: http://127.0.0.1:7440/domain/enduro/workflows/.

## Development workflow

You can use the standard Go build workflows such `go build` or `go install` but
we provide a couple of shortcuts that use our custom build directory and flags:

    # Build the binary in ./build/enduro
    make enduro-dev

    # Build and run
    make run

    # Since `run` is the default, you can just type
    make

If you are looking for a source-code editor, Visual Studio Code has great Go
support. It uses the official language server, [gopls][gopls], which is it not
stable yet - it is recommendable to read their docs and keep it up to date. It

Other useful commands are `make lint` and `make test`. This project uses
GolangCI-Lint, an aggregator of many linters. It is installed with `make tools`.
You can enable it in Visual Studio Code as follows:

```json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"]
}
```


[docker-compose]: https://docs.docker.com/compose/install/
[mc]: https://docs.min.io/docs/minio-client-quickstart-guide.html
[go]: https://golang.org/doc/install
[gopls]: https://github.com/golang/tools/blob/master/gopls/README.md
[yarn]: https://classic.yarnpkg.com/en/docs/install/
[make]: https://www.gnu.org/software/make/
