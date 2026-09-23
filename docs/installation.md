# Installation

## Introduction

This is a work-in-progress document that we expect to extend as we refine the
design of Enduro and we learn more about Temporal. You may prefer to use our
[development environment](./contributing.md#environment) if you only came here
for evaluating purposes.

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
guide][temporal-deployment].

### Other dependencies

#### S3-compatible storage + Redis

Enduro can consume objects uploaded to S3-compatible object storage. Object
storage and Redis can be considered dependencies only when Enduro is set up
with at least one S3 watcher entry.

Enduro does not poll buckets directly. Instead, `[[watcher.s3]]` consumes
object-created events from Redis, downloads the referenced object from the
configured S3-compatible endpoint, and starts the processing workflow.

There are two supported Redis event formats:

- `eventFormat = "enduro"` reads Enduro-normalized object events. This is the
  preferred format for new integrations because provider-specific details are
  handled before events reach the watcher.
- `eventFormat = "minio"` reads native MinIO Redis notification payloads. This
  keeps the legacy MinIO + Redis notification path supported.

The bundled development environment uses SeaweedFS as the S3-compatible storage
service. SeaweedFS sends filer webhook events to Enduro's
`[objectEventWebhook]` adapter, which normalizes object-created file events and
writes them to Redis:

```text
SeaweedFS filer webhook
  -> Enduro [objectEventWebhook]
  -> Redis list object-events
  -> Enduro [[watcher.s3]]
  -> S3 object download
  -> processing workflow
```

In this setup, `[[watcher.s3]]` uses `eventSource = "redis"` and
`eventFormat = "enduro"`. The detailed configuration attributes are described
in the [configuration reference](./configuration-reference.md).

#### Temporal Web UI

Temporal provides a web interface that offers great visibility of workflows and
activities. Compared to the command-line interface, it does not provide as much
control and granularity, but it is useful in simpler use cases.

Visit [Temporal Web UI][temporal-web-ui] to know more.

#### Prometheus

Though not a dependency per se, Prometheus can be used to pull metrics from
both Temporal (be prepared for an extensive set of metrics here) and Enduro.

In Enduro, the `debugListen` configuration parameter determines the address of
the HTTP server from which the metrics are served. E.g. use
`debugListen=127.0.0.1:9001` to make metrics available at
<http://127.0.0.1:9001/metrics>.

## Enduro

Enduro binaries can be found at the [release page][enduro-release-page]. Learn
more about the configuration details
[here](./configuration-reference.md). Review the
[Security Configuration](./security-configuration.md) guide before exposing
Enduro to operators.

Before upgrading, follow the [Enduro upgrade procedure] to ensure that all
Enduro Workflow Executions are closed. By default, Enduro applies pending
database migrations automatically when it starts. See
[Database migrations](./upgrades.md#database-migrations) to manage migrations
manually or before downgrading Enduro to a release with an older database
schema.

### API server

The configuration attribute `api.listen` determines the address where Enduro
sets up its server to listen. This server exposes the API and web interface.

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

### Live updates behind proxies

`/collection/monitor` uses Server-Sent Events (SSE), a long-lived HTTP response
that delivers collection updates. Proxies and security appliances must forward
messages promptly; buffering or inspection that waits for a complete response
can stall delivery. A proxy that supported Enduro's previous WebSocket connection
may still buffer SSE responses. The [Connection monitor] shows whether the
browser is connected and receiving events.

For Nginx, apply these directives in the existing location serving this endpoint,
preserving its upstream, authentication, and forwarding settings:

```nginx
proxy_buffering off;
proxy_cache off;
proxy_read_timeout 60s;  # Already the Nginx default.
```

[`proxy_buffering off`] forwards data as it arrives.
[`proxy_read_timeout`] measures inactivity between upstream
reads, not total connection duration. Enduro's ten-second heartbeats should
keep it open; increasing the timeout does not fix buffering. If creating a
separate location, retain the existing access controls: sibling locations do
not inherit each other's settings.

If compression is enabled for event streams, try [`gzip off;`] to
rule out Nginx's own compression. Other appliances may need separate changes.
Validate with `nginx -t` before reloading.

To check delivery, replace the hostname below and authenticate if required:

```sh
curl --no-buffer --include --max-time 35 --header 'Accept: text/event-stream' https://enduro.example.org/collection/monitor
```

Expect HTTP 200, `Content-Type: text/event-stream`, an immediate `data:` message
containing `"type":"hello"`, then `"type":"ping"` about every ten seconds.
`--no-buffer` disables cURL's output buffering. The command deliberately times out after
35 seconds because the stream stays open.

Where access permits, compare direct backend access with the public address.
Events delayed, batched, or absent only through the proxy suggest a problem in
that access path. In browser network tools, an open request is normal; check
for arriving messages. Buffering can stall delivery without a browser error.

[Connection monitor]: ./user-manual.md#connection-monitor
[`proxy_buffering off`]: https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffering
[`proxy_read_timeout`]: https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_read_timeout
[`gzip off;`]: https://nginx.org/en/docs/http/ngx_http_gzip_module.html#gzip
[multiple ways to run a Temporal Cluster]: https://docs.temporal.io/kb/all-the-ways-to-run-a-cluster
[Ansible role]: https://github.com/artefactual-labs/ansible-enduro-temporal
[temporal-deployment]: https://docs.temporal.io/cluster-deployment-guide
[temporal-web-ui]: https://docs.temporal.io/web-ui
[docker-restart-policy]: https://docs.docker.com/config/containers/start-containers-automatically/#use-a-restart-policy
[enduro-release-page]: https://github.com/artefactual-labs/enduro/releases
[Enduro upgrade procedure]: ./upgrades.md#upgrade-enduro
