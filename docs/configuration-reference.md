# Configuration Reference

Enduro is configured via the `enduro.toml` configuration file which uses TOML,
a well-known file format for configuration files.

The default search paths are `/etc/enduro.toml`, `$HOME/.config/enduro.toml`,
and the current directory. Additionally, users can indicate the configuration
file using the optional argument `--config=example.toml`.

This page is the schema-style reference for configuration attributes and
examples. For operator workflows and concepts, see the
[User Manual](./user-manual.md). For secure deployment guidance, see the
[Security Configuration](./security-configuration.md) guide. For recovery
behavior and limitations, see the [Recovery Guide](./user-manual-recovery.md).

## Root

Main configuration attributes that do not belong to a specific section.

#### `verbosity` (Int)

Chattiness of log operations. `0` is the default and best for production.

E.g.: `0`

#### `debug` (Bool)

When enabled, the application logger will be configured with a human-readable
format and a colored log record formatter.

E.g.: `false`

#### `debugListen` (String)

Address of the debugging HTTP server including Prometheus metrics and profiling
data.

E.g.: `"127.0.0.1:9001"`

## `[telemetry]`

Telemetry configuration details.

### `[telemetry.traces]`

Tracing configuration.

For example:

```toml
[telemetry.traces]
enabled = false
address = "127.0.0.1:12345"
ratio = 1.0
```

#### `enabled` (Bool)

When enabled, traces will be delivered to the tracing data collector. It is
disabled by default.

E.g.: `false`

#### `address` (String)

Address of the OpenTelemetry tracing data collector (gRPC), e.g. Grafana Agent
or OpenTelemetry Collector.

E.g.: `"127.0.0.1:12345"`

#### `ratio` (Float64)

Sampling ratio. A sampling ratio of 0.25 means that, on average, one out of
every four traces will be sampled. The default is to sample every transfer.

E.g.: `0.25`

### `[telemetry.metrics]`

Not available yet.

## `[temporal]`

Connection details with the Temporal server.

#### `namespace` (String)

Name of the Temporal namespace used by this application.

E.g.: `"enduro"`

#### `address` (String)

Address of the Temporal front-end server.

E.g.: `"127.0.0.1:7400"`

## `[api]`

Configuration of the Enduro API server.

#### `listen` (String)

Address of the primary Enduro server. It serves the API and the new Nuxt web UI.

E.g.: `"127.0.0.1:9000"`

#### `legacyListen` (String)

Optional address of a second Enduro server that serves the API and the legacy
web UI. Leave this unset unless operators still need access to the old
interface alongside the new default frontend.

E.g.: `"127.0.0.1:9002"`

#### `debug` (Boolean)

When enabled, the logger used in the API layer will produce log entries with
detailed information about incoming requests and outgoing responses, headers,
parameters and bodies.

E.g.: `false`

#### `allowedOrigins` (Array of strings)

Additional origins allowed by CORS and by Enduro's browser cross-origin write
protection, which uses Go's `net/http.CrossOriginProtection`. This CSRF-style
control rejects unsafe cross-origin browser requests before they can reach
state-changing API handlers.

Safe methods such as `GET`, `HEAD`, and `OPTIONS` are always allowed through
this protection, but CORS response headers are removed for origins that are not
same-origin or listed here. Requests without browser origin signals and requests
whose origin matches the request host are allowed.

Origins must match the browser `Origin` header exactly, including scheme and
port when present, for example `https://dashboard.example.org` or
`http://localhost:3000`.

The supported modes are:

```toml
# Default open CORS mode.
# This permits CORS responses for any origin and disables Enduro's browser
# cross-origin write protection. Avoid this mode when browsers authenticate to
# Enduro with cookies or other automatically attached credentials.
allowedOrigins = ["*"]
```

```toml
# Recommended same-origin deployment.
# Use this when Enduro serves the dashboard and API from the same origin,
# including when both are protected by the same external access-control layer.
allowedOrigins = []
```

```toml
# Split-origin deployment.
# Use this when the dashboard and API are intentionally served from different
# origins.
allowedOrigins = ["https://dashboard.example.org"]
```

See [Security Configuration](./security-configuration.md) for deployment
guidance and the CSRF rationale for browser-accessible APIs.

#### `contentSecurityPolicy` (String)

Value of the `Content-Security-Policy` response header. When set, Enduro also
sends `X-Content-Type-Options`, `Referrer-Policy`, and `X-Frame-Options`.

The supported modes are:

```toml
# Default when contentSecurityPolicy is omitted.
contentSecurityPolicy = ""
```

```toml
# Recommended starting point for same-origin dashboard/API deployments.
contentSecurityPolicy = "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; connect-src 'self'; img-src 'self' data:; font-src 'self' data:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
```

The example policy is compatible with the bundled dashboard and legacy UI while
limiting resource loading to the same origin. Deployments that intentionally
load assets or connect to APIs on other origins should extend the relevant CSP
directives before enabling it.

Enduro does not enable a default CSP yet because the policy must remain
compatible with both the new dashboard and the legacy Vue 2 UI. We intend to
define and enable stronger default browser security headers after the legacy UI
is phased out and the new dashboard can be validated against the stricter
policy.

E.g.: `"default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; connect-src 'self'; img-src 'self' data:; font-src 'self' data:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"`

## `[database]`

Database connection details.

#### `dsn` (String)

Connection details using the [Data Source Name format].

E.g.: `enduro:enduro123@tcp(127.0.0.1:7450)/enduro`

## `[watcher]`

Watchers monitor data sources like filesystems or S3 buckets to process new
objects.

### `[[watcher.filesystem]]`

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
ignore = '(^\.gitkeep)|(^*\.mft)$'

# Omit the top-level directory of the transfer after extraction.
stripTopLevelDir = true

# Reject transfers with duplicate transfer names.
rejectDuplicates = false

# Exclude hidden files from transfer.
excludeHiddenFiles = false

# Archivematica transfer type.
transferType = "standard"
```

Namely, it monitors the `watched-dir` directory. It uses the inotify API for
better efficiency. New objects will be processed using the `am` pipeline,
defined elsewhere in the configuration document. The workflow will wait for
eleven seconds before the original object is removed from the filesystem. After
extraction, the top-level directory will be omitted.

#### `retentionPeriod` (String)

Specifies the duration for which a transfer will be retained before removal.
This attribute is mutually exclusive with completedDir. If undefined, the
transfer is not removed. If set to '0s', the transfer is removed immediately.
This option is undefined by default, meaning the transfer will not be removed
unless specified otherwise.

The string should be constructed as a sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "30m", "24h" or "2h30m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

E.g.: `"10m"`

#### `rejectDuplicates` (Boolean)

When enabled, the workflow will execute a check on the internal database for
successfully completed transfers with the same transfer name as the currently
processing package. If it finds a duplicate the transfer will fail.

E.g.: `false`

#### `excludeHiddenFiles` (Boolean)

When enabled, the workflow will exclude hidden files from the transfer.

E.g.: `false`

#### `transferType` (String)

Archivematica submission transfer type.

The string should be one of the following:
"standard", "zipfile", "unzipped bag", "zipped bag", "dspace", "maildir",
"TRIM" or "dataverse".

E.g.: `"standard"`

#### `completedDir` (String)

The path where transfers are moved into when processing has completed
successfully. Not available in other type of watchers. It is mutually
exclusive with `retentionPeriod`.

E.g.: `"/mnt/landfill"`

#### `ignore` (String)

A regular expression or search pattern that can be used to ignore new files or
directories identified by the filesystem watcher. Its main purpose is to
provide a mechanism to avoid launching the processing workflow for new entries
that may not be fully transferred by the time that they're identified.

For example, when transferring a directory, the watcher will likely start the
processing workflow before all the contents are fully transferred. Using a
pattern like `'^*\.ignored$'` allows for transferring a directory (e.g.
`sample-transfer.ignored`) that will not be processed by the system until the
user renames it to have the `.ignored` suffix removed.

The expressions must use the [RE2 syntax].

E.g.: `'^*\.ignored$'`

#### `pipeline` (String | Array(String))

The name of the pipeline to be used during processing. If undefined, one will
be chosen randomly from the list of existing pipelines configured. If an Array
is provided, the name will be chosen randomly from its values.

E.g.: `"am"`, `["am1", "am2"]`

### `[[watcher.minio]]`

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

# Reject transfers with duplicate transfer names.
rejectDuplicates = false

# Exclude hidden files from transfer.
excludeHiddenFiles = false

# Archivematica transfer type.
transferType = "standard"
```

MinIO will deliver new events to us via a Redis instance at
`redis://127.0.0.1:7470` using a list named `minio-events`. When a new event is
received, the object will be downloaded from `http://127.0.0.1:7460`, the
MinIO server, using path-style URLs and region `us-west-1`, bucket `sips`.
Attributes `key` and `secret` are for authentication. After extraction of the
object, the top-level directory will be omitted. All objects will be processed
using the `am` pipeline.

#### `retentionPeriod` (String)

Specifies the duration for which a transfer will be retained before removal.
This attribute is mutually exclusive with completedDir. If undefined, the
transfer is not removed. If set to '0s', the transfer is removed immediately.
This option is undefined by default, meaning the transfer will not be removed
unless specified otherwise.

The string should be constructed as a sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "30m", "24h" or "2h30m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

E.g.: `"10m"`

#### `rejectDuplicates` (Boolean)

When enabled, the workflow will execute a check on the internal database for
successfully completed transfers with the same transfer name as the currently
processing package. If it finds a duplicate the transfer will fail.

E.g.: `false`

#### `excludeHiddenFiles` (Boolean)

When enabled, the workflow will exclude hidden files from the transfer.

E.g.: `false`

#### `transferType` (String)

Archivematica submission transfer type.

The string should be one of the following:
"standard", "zipfile", "unzipped bag", "zipped bag", "dspace", "maildir",
"TRIM" or "dataverse"

E.g.: `standard`

#### `pipeline` (String | Array(String))

The name of the pipeline to be used during processing. If undefined, one will
be chosen randomly from the list of existing pipelines configured. If an Array
is provided, the name will be chosen randomly from its values.

E.g.: `"am"`, `["am1", "am2"]`

## `[pipeline]`

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
unbag = false
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

Path where Enduro prepares transfers before submitting them to Archivematica.
Without a transfer publisher, this must be the Archivematica transfer source
directory and must be locally accessible to Enduro. With a transfer publisher,
this is a local staging directory used by Enduro before the transfer is copied
to the external transfer source.

E.g.: `"~/.am/ss-location-data"`

#### `[pipeline.transferPublisher]`

Optional settings that publish prepared transfers to an Archivematica-visible
transfer source before Enduro submits the transfer to Archivematica. When this
section is omitted, Enduro keeps the existing behavior: it submits the path
created inside `transferDir` directly.

The initial publisher type is `sftp`. This is useful when Enduro can prepare
transfers locally but Archivematica reads transfers from a filesystem that is
only available through SFTP, such as ambox's SFTPGo transfer source.

```toml
[[pipeline]]
name = "ambox"
baseURL = "http://ambox:64080"
user = "test"
key = "test"
transferDir = "/var/lib/enduro/transfers"
processingConfig = "automated"
storageServiceURL = "http://test:test@ambox:64081"

[pipeline.transferPublisher]
type = "sftp"
host = "ambox"
port = 64022
user = "archivematica"
password = "12345"
remoteDir = "/"
submittedPathPrefix = "archivematica/transfers"
insecureIgnoreHostKey = true
```

For this ambox example, Enduro uploads the prepared transfer to the SFTP user's
root directory. ambox maps that SFTP root to
`/home/archivematica/transfers`, while Archivematica's default transfer source
is `/home`. The `submittedPathPrefix` therefore makes Enduro submit paths such
as `archivematica/transfers/<transfer>` to Archivematica. After the workflow
finishes or fails, Enduro attempts to clean up the published SFTP transfer just
as it cleans up locally prepared transfers.

##### `type` (String)

Publisher implementation. Supported value: `"sftp"`.

##### `host` (String)

SFTP server hostname or IP address.

##### `port` (Integer)

SFTP server port. Defaults to `22` when omitted or set to `0`.

##### `user` (String)

SFTP username.

##### `password` (String)

SFTP password. Either `password` or `privateKey.path` must be configured.

##### `[pipeline.transferPublisher.privateKey]`

Optional private key authentication settings. When `privateKey.path` is set,
Enduro uses SSH public key authentication instead of password authentication.

```toml
[pipeline.transferPublisher.privateKey]
path = "/var/lib/enduro/ssh/id_ed25519"
passphrase = ""
```

##### `path` (String)

Path to the SSH private key used to authenticate with the SFTP server.

##### `passphrase` (String)

Optional passphrase used to decrypt the private key.

##### `remoteDir` (String)

Remote directory where Enduro uploads transfers. Defaults to `"/"`. The
transfer path created under `transferDir` is appended to this directory.

##### `submittedPathPrefix` (String)

Path prefix to prepend to the transfer path submitted to Archivematica. Use
this when the SFTP-visible path differs from the path visible from
Archivematica's Transfer Source location.

##### `hostKey` (String)

Expected SSH host public key in authorized-key format. Use this in production
when the SFTP server host key is stable.

##### `knownHostsFile` (String)

Path to an OpenSSH `known_hosts` file. This can be used instead of `hostKey`.

##### `insecureIgnoreHostKey` (Boolean)

Disable SSH host key verification. This is convenient for disposable local or
CI environments with ephemeral SFTP host keys, but it should not be used for
production deployments.

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

#### `[pipeline.recovery]`

Optional settings that make retry check Storage Service for the package
Archivematica already created for a collection before Enduro starts a new
transfer.

Recovery is disabled by default. Configuring `storageServiceURL` alone does not
change retry behavior. Enduro only uses the recovery flow when
`reconcileExistingAIP = true` is set for that pipeline. This is mainly for
cases where Archivematica ingest fails during or after Storage Service
replication, Enduro records the collection as `error`, but the source AIP may
already exist in Storage Service, so an immediate full retry could create a
duplicate AIP and duplicate replicas. With recovery enabled, retry first
reconciles that known package in Storage Service and only starts a fresh
transfer if Storage Service shows the package is missing. See the
[Recovery Guide](./user-manual-recovery.md) for settings, examples, workflow
behavior, operator-facing details, and current limitations. In the current
implementation, temporary post-ingest reconciliation retries still occupy a
pipeline slot until Storage Service confirms the outcome or the retry window
expires, so recovery can temporarily reduce effective throughput on busy
pipelines.

At a high level:

- omit `requiredLocations`, or set it to `[]`, when the primary package alone
  is enough to satisfy the storage rule
- set `requiredLocations` to one or more Storage Service location UUIDs when
  those locations must also be confirmed before the collection is considered
  complete

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

#### `unbag` (Boolean)

If enabled, bagged transfers will be unbagged.

E.g.: `false`

## `[validation]`

#### `checksumsCheckEnabled` (String)

If enabled, this validator will stop the workflow with an error if the
transfer does not include a document with checksums, e.g. `checksum.sha1`.

E.g.: `false`

## `[worker]`

#### `heartbeatThrottleInterval` (String)

Specifies the interval at which Enduro sends heartbeats to the workflow engine.

The string should be constructed as a sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "30m", "24h" or "2h30m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

E.g.: `5m` (String)

#### `maxConcurrentWorkflowsExecutionsSize` (int)

Sets the maximum number of concurrent workflow executions that Enduro will
accept from the workflow engine.

A good rule of thumb is to set this value to twice the sum of the capacities of
all the configured pipelines. For example, if two pipelines are configured with
a capacity of `5` each, the value should be `20`.

E.g.: `10`

#### `maxConcurrentSessionExecutionSize` (int)

Sets the maximum number of concurrently running (workflow engine) sessions that
Enduro supports. This value governs how many concurrent SIPs are going to be
processed at any given time, regardless of pipeline capacity. This setting can
be used to throttle from a single place how many concurrent pipelines Enduro
will run.

We recommend setting this value to be directly proportional to (or higher than)
the Archivematica pipeline capacity.

E.g.: `5`

## `[workflow]`

#### `activityHeartbeatTimeout` (String)

Specifies the timeout duration for activities that send heartbeats to the
workflow engine. If the activity takes more time to send a heartbeat to the
workflow engine, the workflow will fail with a `heartbeatTimeout` error.

The string should be constructed as a sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "30m", "24h" or "2h30m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

E.g.: `5m` (String)

## Configuration example

See https://github.com/artefactual-labs/enduro/blob/main/enduro.toml.

[Data Source Name format]: https://github.com/go-sql-driver/mysql#dsn-data-source-name
[RE2 syntax]: https://github.com/google/re2/wiki/Syntax
