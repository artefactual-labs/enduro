---
title: "Installation"
linkTitle: "Installation"
weight: 2
description: >
  Follow this guide to install Enduro and its dependencies.
---

{{% alert title="Warning" color="warning" %}}
Enduro is still at its early stages. **Use with caution!**
{{% /alert %}}

## Introduction

This is a work-in-progress document that we expect to extend as we refine the
design of Enduro and we learn more about Cadence. You may prefer to use our
[development environment][enduro-devenv] if you only came here for evaluating
purposes.

The instructions below are not specific to a particular environment.

## Dependencies

Enduro's two main dependencies are MySQL and Cadence.

### MySQL

We use MySQL 5.7. MySQL 8.x has not been tested yet but there is at least one
known incompatibility in Cadence.

It is recommendable to change the default character settings of the server. For
example, it can be added to the `[mysqld]` configuration:

```ini
[mysqld]
character-set-server=utf8mb4
collation-server=utf8mb4_general_ci
```

### Cadence

Cadence is the orchestration engine. Currently, Cadence is only distributed as a
Docker image. It is possible to extract the static binaries from the image using
the following commands:

    mkdir -p $HOME/cadence
    cd $HOME/cadence
    ctid=$(docker create ubercadence/server:0.9.4)
    docker cp ${ctid}:/usr/local/bin/cadence .
    docker cp ${ctid}:/usr/local/bin/cadence-server .
    docker cp ${ctid}:/usr/local/bin/cadence-sql-tool .
    docker cp ${ctid}:/etc/cadence/schema/mysql/v57/cadence/versioned ./cadence-migrations
    docker cp ${ctid}:/etc/cadence/schema/mysql/v57/visibility/versioned ./visibility-migrations

A detailed description of what we are extracting from the Docker image:

* `cadence` is the [Cadence CLI management tool][cadence-cli]. It is the most
  important tool for Cadence administrators.
* `cadence-server` is the server binary.
* `cadence-sql-tool` is the database migration tool which we're going to use
  later in this document to apply migrations. The `cadence-migrations` and
  `visibility-migrations` directories contain the actual SQL migrations.

#### Running Cadence

This is an example on how to run the Cadence server:

    cadence-server \
      --root=/etc/cadence \
      --env=demo \
      --services=frontend,matching,history,worker \
        start

The previous command will read config from `/etc/cadence/config/demo.yaml`.
Unfortunately, the configuration is not documented properly. Please use our
[development config][development-config] as a reference.

The `--services` flag is telling Cadence which subsystems to load. It allows
administrators to set up complex distributed deployments. It is safe to start
with a simpler configuration with the four main services together.

Off all the services that make up Cadence, `frontend` is the one we need to
connect. By default, it listens on `127.0.0.1:7933` but our development
environment uses `127.0.0.1:7400` instead.

#### Database setup

Cadence needs two MySQL databases set up properly before starting the daemon
(`cadence` and `cadence_visibility`). Enduro needs its own MySQL database
(`enduro`). The following snippet provisions the databases as needed:

```mysql
CREATE USER 'enduro'@'%' IDENTIFIED BY 'enduro123';
CREATE DATABASE IF NOT EXISTS cadence CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON cadence.* TO 'enduro'@'%';
CREATE DATABASE IF NOT EXISTS cadence_visibility CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON cadence_visibility.* TO 'enduro'@'%';
CREATE DATABASE IF NOT EXISTS enduro CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON enduro.* TO 'enduro'@'%';
```

Another required step is to apply the SQL migrations using `cadence-sql-tool`.
We do similarly in the development environment via [`seed.sh`][cadence-dbseed]
which can be used as a reference.

### Other dependencies

#### MinIO + Redis

Enduro can consume objects uploaded to MinIO by listening to [MinIO events via
Redis][minio-redis-access]. MinIO and Redis can be considered dependencies only
when Enduro is set up with at least one ``watcher.minio`` entry.

#### Cadence Web

Cadence provides a web interface that offers some visibility of workflows and
activities. Compared to the command-line interface, it does not provide as much
control and granularity, but it is useful in simpler use cases.

It is distributed as a Docker image but it can also be installed via Node
package managers such npm or Yarn. With Docker installed, you can run it with:

    docker run \
      --env=CADENCE_WEB_PORT=10000 \
      --env=CADENCE_TCHANNEL_PEERS=127.0.0.1:7933 \
        ubercadence/web:3.4.1

If you're deploying with Docker, use  `--restart=always` or similar to apply a
[restart policy][docker-restart-policy] to the container. At the moment, the
process ignores system signals. Use `docker container kill` if you want to stop
the container.

#### Prometheus

Though not a dependency per se, Prometheus can be used to pull metrics from
both Cadence (be prepared for an extensive set of metrics here) and Enduro.

In Enduro, the `debugListen` configuration parameter determines the address of
the HTTP server from which the metrics are served. E.g. use
`debugListen=127.0.0.1:9001` to make metrics available at
http://127.0.0.1:9001/metrics.

## Enduro

Enduro binaries can be found at the [release page][enduro-release-page].

The only configuration input accepted at the moment is the TOML configuration
file. Use `enduro --config=config.toml` or rely one one of the default
configuration locations: `/etc/enduro.toml` or `$HOME/.config/enduro.toml`, as
well as in the current directory.

The [development environment configuration][enduro-config] can be used as a
reference. The sections and attributes are not entirely self-explanatory. We're
hoping to add individual descriptions at some point. It is very likely that the
configuration structure changes over time as we are still figuring things out in
the proof of concept.

### Domain creation

Enduro is set up to employ a specific [Cadence domain][cadence-domain] which
needs to be created beforehand with Cadence's CLI tool. The following is an
example that creates the `enduro` domain assuming that Cadence's `frontend` is
listening on `127.0.0.1:7933`.

    docker run -it --network=host --rm \
      ubercadence/cli:master --address=127.0.0.1:7933 --domain=enduro domain register

### API server

The configuration attribute `api.listen` determines the address where Enduro
sets up the server to listen.

Assuming that `api.listen=127.0.0.1:9000`, opening http://127.0.0.1:9000 from
your browser will bring you to the web interface. An example on how to consume
the API via cURL is `curl -Ls 127.0.0.1:9000/collection | jq`:

```json
[
  {
    "id": 1,
    "name": "DPJ-SIP-35823fa7-07fe",
    "status": "done",
    "workflow_id": "processing-workflow-5a4b899d-6a17-4a95-b403-749d9c4f1e81",
    "run_id": "0306c1e1-c0ef-4ed3-aa44-74936e34c3e4",
    "transfer_id": "fb55b95e-2395-4a95-972e-c7273fca6815",
    "aip_id": "ec1526c9-acad-4f78-8422-3c3a0a4c5de3",
    "original_id": "35823fa7-07fe-48a8-a1d1-5d8cb9bd097e",
    "created_at": "2019-10-14T01:40:48Z",
    "completed_at": "2019-10-14T01:41:13Z"
  },
  {
    "id": 2,
    "name": "DPJ-SIP-b78e419d-2c7f",
    "status": "done",
    "workflow_id": "processing-workflow-26142269-3da1-49da-8a32-529348f73fe3",
    "run_id": "b7e25ba7-7d4e-4815-867e-f14dccc34e13",
    "transfer_id": "25f99f5b-8f16-411d-913c-6560fa8c5200",
    "aip_id": "033b84d8-a63b-40c9-a91c-a7bc1b9331c4",
    "original_id": "b78e419d-2c7f-4b4a-b5a9-4cdbf0dc3cd4",
    "created_at": "2019-10-16T01:04:44Z",
    "completed_at": "2019-10-16T01:07:22Z"
  }
]
```


[cadence-deployment]: https://github.com/uber/cadence/tree/master/docker
[cadence-dbseed]: https://github.com/artefactual-labs/enduro/blob/main/hack/cadence/seed.sh
[cadence-cli]: https://cadenceworkflow.io/docs/08_cli
[cadence-domain]: https://cadenceworkflow.io/docs/04_glossary#domain
[development-config]: https://github.com/artefactual-labs/enduro/blob/main/hack/cadence/config.yml
[minio-redis-access]: https://docs.min.io/docs/minio-bucket-notification-guide.html#Redis
[docker-restart-policy]: https://docs.docker.com/config/containers/start-containers-automatically/#use-a-restart-policy
[enduro-release-page]: https://github.com/artefactual-labs/enduro/releases
[enduro-config]: https://github.com/artefactual-labs/enduro/blob/main/enduro.toml
[enduro-devenv]: {{< ref "/docs/development/environment" >}}
