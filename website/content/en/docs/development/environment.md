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
makes thing much simpler.

There are two main dependencies: [Go][go] and [Docker Compose][docker-compose].
Follow the links if you want to know how to install them. Use Go 1.13 or newer.

Spin up the environment with the following command:

    docker-compose up -d

Cadence will crash right away because the database has not been set up properly.
Run the following command to introduce the MySQL tables needed by Cadence:

    make seed

After a few seconds, Cadence should be up again. Since Cadence is a multitenant
service, we need to create a Cadence domain for Enduro:

    make domain

## Run your first test

In the environment that we've just build, Enduro monitors the pre-configured
blobstore so it can react to files uploded by the user.

Let's give it a try. First, install and set up Minio's CLI:

    mc config host add enduro http://127.0.0.1:7460 "36J9X8EZI4KEV1G7EHXA" "ECk2uqOoNqvtJIMQ3WYugvmNPL_-zm3WcRqP5vUM"

And upload a file into the `sips` bucket:

    curl -s https://news.ycombinator.com/y18.gif | mc pipe enduro/sips/y18.gif

Now compile and run Enduro:

    go install && enduro

Open Cadence Web: http://127.0.0.1:7440/domain/enduro/workflows/.

## Development workflow

In most cases, you can build the application with `go build`. The executable
file is located in the root directory.

If you are looking for a source-code editor, Visual Studio Code has great Go
support. It uses the official language server, [gopls][gopls], which is it not
stable yet - it is recommendable to read their docs and keep it up to date.

Run `make tools` to install some other local dependencies that you are going to
need.

This project uses GolangCI-Lint, a linters aggregator. Install it using the
example above. Run it with `golangci-lint run` or set it up in your editor.
This is how you can configure Visual Studio Code to use it:

```json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"]
}
```


[docker-compose]: https://docs.docker.com/compose/install/
[go]: https://golang.org/doc/install
[gopls]: https://github.com/golang/tools/blob/master/gopls/README.md
