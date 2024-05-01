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
design of Enduro and we learn more about Temporal. You may prefer to use our
[development environment][enduro-devenv] if you only came here for evaluating
purposes.

The instructions below are not specific to a particular environment.

## Dependencies

Enduro's two main dependencies are MySQL and Temporal.

### MySQL

We use MySQL 8, which serves as the data store for Enduro and Temporal.

### Temporal

Temporal is the orchestration engine. There are
[multiple ways to run a Temporal Cluster]. Our standard configuration uses an
[Ansible role] to deploy both Enduro and Temporal on a remote server.

More on this topic can be found at the official [Temporal Cluster deployment
guide](temporal-deployment).

### Other dependencies

#### MinIO + Redis

Enduro can consume objects uploaded to MinIO by listening to [MinIO events via
Redis][minio-redis-access]. MinIO and Redis can be considered dependencies only
when Enduro is set up with at least one `watcher.minio` entry.

#### Temporal Web UI

Temporal provides a web interface that offers great visibility of workflows and
activities. Compared to the command-line interface, it does not provide as much
control and granularity, but it is useful in simpler use cases.

Visit [Temporal Web UI](temporal-web-ui) to know more.

#### Prometheus

Though not a dependency per se, Prometheus can be used to pull metrics from
both Temporal (be prepared for an extensive set of metrics here) and Enduro.

In Enduro, the `debugListen` configuration parameter determines the address of
the HTTP server from which the metrics are served. E.g. use
`debugListen=127.0.0.1:9001` to make metrics available at
<http://127.0.0.1:9001/metrics>.

## Enduro

Enduro binaries can be found at the [release page][enduro-release-page]. Learn
more about the configuration details [here]({{< relref
"/docs/user-manual/configuration" >}}).

### API server

The configuration attribute `api.listen` determines the address where Enduro
sets up the server to listen.

Assuming that `api.listen=127.0.0.1:9000`, opening <http://127.0.0.1:9000> from
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

[multiple ways to run a Temporal Cluster]: https://docs.temporal.io/kb/all-the-ways-to-run-a-cluster
[Ansible role]: https://github.com/artefactual-labs/ansible-enduro-temporal
[temporal-deployment]: https://docs.temporal.io/cluster-deployment-guide
[temporal-web-ui]: https://docs.temporal.io/web-ui
[minio-redis-access]: https://docs.min.io/docs/minio-bucket-notification-guide.html#Redis
[docker-restart-policy]: https://docs.docker.com/config/containers/start-containers-automatically/#use-a-restart-policy
[enduro-release-page]: https://github.com/artefactual-labs/enduro/releases

[enduro-devenv]: {{< ref "/docs/development/environment" >}}
