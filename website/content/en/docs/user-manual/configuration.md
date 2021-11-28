---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
description: >
  Configuration options available in Enduro.
---

## Configuration

Enduro is configured via the ``enduro.toml`` configuration file which uses TOML,
a well-known file format for configuration files.

The default search paths are `/etc/enduro.toml`, `$HOME/.config/enduro.toml`,
and the current directory. Additionally, users can indicate the configuration
file using the optional argument `--config=example.toml`.

**This is work in progress.**

## Sections

### `[cadence]`

Connection details with the Cadence server.

#### `domain` (String)

Name of the Cadence domain used by this application.

E.g.: `"enduro"`

#### `address` (String)

Address of the Cadence front-end server.

E.g.: `"127.0.0.1:7400"`

### `[api]`

Configuration of the Enduro API server.

#### `listen` (String)

Address of the Enduro API server.

E.g.: `"127.0.0.1:9000"`

#### `debug` (Boolean)

When enabled, loggers will be configured with increased verbosity.

E.g.: `false`

### `[database]`

Database connection details.

#### `dsn` (String)

Connection details using the [Data Source Name format].

E.g.: `enduro:enduro123@tcp(127.0.0.1:7450)/enduro`

### `[watcher]`

Watchers monitor data sources like filesystems or S3 buckets to process new objects.

#### `[[watcher.filesystem]]`

The following watcher monitors the filesystem:

```toml
[[watcher.filesystem]]

# Name of the watcher.
name = "dev-fs"

# Monitoring path.
path = "./hack/watched-dir"

# Use of the inotify API for efficiency - not always available.
inotify = true

# Defined elsewhere, this is the name of the Archivematica pipeline used.
pipeline = "am"

# Wait for 11 seconds before removing the object from the filesystem.
retentionPeriod = "11s"

# Ignore new filesystem entries matching the following pattern.
ignore = '(^\.)|(^\.gitkeep)|(^*\.mft)$'

# Omit the top-level directory of the transfer after extraction.
stripTopLevelDir = true
```

Namely, it monitors the `watched-dir` directory. It uses the inotify API for
better efficiency. New objects will be processed using the `am` pipeline,
defined elsewhere in the configuration document. The workflow will wait for
eleven seconds before the original object is removed from the filesystem. After
extraction, the top-level directory will be omitted.

#### `[[watcher.minio]]`

The following monitor watches a MinIO bucket:

```toml
[[watcher.minio]]

# Name of the watcher.
name = "dev-minio"

# Redis server and list used by MinIO to deliver events.
redisAddress = "redis://127.0.0.1:7470"
redisList = "minio-events"

# MinIO server endpoint and other connection details, e.g. name of the bucket.
endpoint = "http://127.0.0.1:7460"
pathStyle = true
key = "minio"
secret = "minio123"
region = "us-west-1"
bucket = "sips"

# Defined elsewhere, this is the name of the Archivematica pipeline used.
pipeline = "am"

# Wait for 10 seconds before removing the object from the bucket.
retentionPeriod = "10s"

# Omit the top-level directory of the transfer after extraction.
stripTopLevelDir = true
```

MinIO will deliver new events to us via a Redis instance at
`redis://127.0.0.1:7470` using a list named `minio-events`. When a new event is
received, the object will be downloaded from `http://127.0.0.1:7460`, the
MinIO server, using path-style URLs and region `us-west-1`, bucket `sips`.
Attributes `key` and `secret` are for authentication. After extraction of the
object, the top-level directory will be omitted. All objects will be processed
using the `am` pipeline.

### `[pipeline]`

Used to define Archivematica pipelines. For example:

```toml
[[pipeline]]
name = "am"
baseURL = "http://127.0.0.1:62080"
user = "test"
key = "test"
transferDir = "~/.am/ss-location-data"
processingDir = ""
processingConfig = "automated"
transferLocationID = "88f6b517-c0cc-411b-8abf-79544ce96f54"
storageServiceURL = "http://test:test@127.0.0.1:62081"
capacity = 3
retryDeadline = "10m"
transferDeadline = "24h"
```

#### `name` (String)

Internal name of the Archivematica pipeline. It is used by watchers to indicate
the pipeline destination.

E.g.: `"am"`

#### `baseURL` (String)

Base URL of the Archivematica web server.

E.g.: `"http://127.0.0.1:62080"`

#### `user` (String)

Archivematica user.

E.g.: `"test"`

#### `key` (String)

Archivematica API key.

E.g.: `"test"`

#### `transferDir` (String)

Path of the transfer source directory (must be locally accessible).

E.g.: `"~/.am/ss-location-data"`

#### `processingDir` (String)

Enduro internal processing directory. Leave empty if unsure.

E.g.: `""`

#### `processingConfig`

Name of the processing configuration to be used when starting a transfer.

E.g.: `"automated"`

#### `transferLocationID`

Identifier of the Transfer Source Location (Storage Service). Optional.

E.g.: `"88f6b517-c0cc-411b-8abf-79544ce96f54"`

#### `storageServiceURL`

URL of the Archivematica Storage Service. It must include the username and API
key in the URL-encoded format suggested below.

E.g.: `"http://test:test@127.0.0.1:62081"`

#### `capacity` (Integer)

The number of active workflows using this pipeline. Enduro attempts to control
the number of concurrent workflows that interact with a pipeline at any given
time.

E.g.: `3`.

#### `retryDeadline` (String)

If present, it sets the amount of time that Enduro waits before abandoning a
transfer when the Archivematica API is returning potentially transient errors
repeatedly, e.g., HTTP 5xx server errors or network connectivity errors.

E.g.: `"10m"`.

#### `transferDeadline` (String)

If present, transfers in this pipeline will be automatically abandoned when the
deadline is exceeded, causing the workflow to complete with an error. It yields
the pipeline slot.

The string should be constructed as a sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "30m", "24h" or "2h30m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

E.g.: `24h` (String)

### `[validation]`

#### `checksumsCheckEnabled` (String)

If enabled, this validator will stop the workflow with an error if the transfer
does not include a document with checksums, e.g. `checksum.sha1`.

E.g.: `false`

## Configuration example

Source: https://github.com/artefactual-labs/enduro/blob/main/enduro.toml.

{{< config >}}

[Data Source Name format]: https://github.com/go-sql-driver/mysql#dsn-data-source-name
