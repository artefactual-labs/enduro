# Containers

Legacy Enduro's release configuration builds Linux containers for AMD64 and
ARM64 from the same source revision as the release archive. A single image
reference selects the correct architecture automatically:

```text
ghcr.io/artefactual-labs/enduro:v<major>.<minor>.<patch>
```

The release workflow publishes the exact Git tag, including prerelease tags. It
does not publish `latest`, major, or minor aliases. Use a release tag when
selecting a version and an image digest when an immutable deployment reference
is required. The existing Linux AMD64 archive remains available with its
original layout; ARM64 is added for containers only.

The image includes the Enduro binary, embedded web interface and database
migrations, CA certificates, timezone data, and license. It uses the same pinned
Chainguard static base as depbot and runs as UID/GID `65532:65532`. There is no
shell or package manager. Enduro is the entrypoint and receives container stop
signals directly.

## Run a release image

The image contains Enduro only. Provide reachable MySQL and Temporal services,
and the Archivematica, watcher, and other services required by your
configuration. Prepare a deployment configuration using the
[configuration reference] and [security guide]. The repository's root
`enduro.toml` is for local development and should not be used unchanged in a
deployment.

Mount your configuration at `/etc/enduro/enduro.toml`, the default configuration
path in the image. In particular:

- Set `api.listen` to `0.0.0.0:9000` so the API and web interface accept traffic
  from outside the container.
- Use service DNS names or reachable addresses for MySQL, Temporal, Redis,
  Archivematica, and object storage. `127.0.0.1` refers to the Enduro container.
- Use absolute paths inside the container for watched files, processing,
  transfers, hooks, and any private keys or known-hosts files.
- Make each writable mount accessible to UID/GID `65532:65532`, and make the
  configuration and any mounted secrets readable by that user. Choose
  permissions appropriate for the deployment rather than making credentials
  world-readable.

For example, these settings illustrate container addresses and paths; they are
only part of a deployment configuration:

```toml
[api]
listen = "0.0.0.0:9000"

[database]
dsn = "enduro:REPLACE_ME@tcp(mysql:3306)/enduro"

[temporal]
address = "temporal:7233"
namespace = "default"
taskQueue = "global"

[[pipeline]]
name = "am"
baseURL = "http://archivematica:8000"
user = "REPLACE_ME"
key = "REPLACE_ME"
transferDir = "/var/lib/enduro/transfers"
processingDir = "/var/lib/enduro/processing"
processingConfig = "automated"
```

Create the processing and transfer directories before starting Enduro. Without a
[transfer publisher], the transfer directory must also be accessible to
Archivematica with the path mapping required by its transfer source. A private
container volume alone does not satisfy that requirement. An SFTP transfer
publisher can instead copy prepared transfers to a remote transfer source.

Once the configuration, writable directories, and service network exist, a
Docker invocation can look like this. Replace the image tag and host paths for
your deployment:

```sh
docker run --rm --name enduro \
  --network enduro \
  --publish 127.0.0.1:9000:9000 \
  --mount type=bind,src=/srv/enduro/enduro.toml,dst=/etc/enduro/enduro.toml,readonly \
  --mount type=bind,src=/srv/enduro/processing,dst=/var/lib/enduro/processing \
  --mount type=bind,src=/srv/enduro/transfers,dst=/var/lib/enduro/transfers \
  --read-only --tmpfs /tmp:rw,nosuid,nodev,mode=1777 \
  --cap-drop ALL --security-opt no-new-privileges \
  ghcr.io/artefactual-labs/enduro:vX.Y.Z
```

The example binds the host port to loopback; expose Enduro through your
configured reverse proxy and authentication setup. `/tmp` is required for
temporary files. If `processingDir` is empty, Enduro also uses the temporary
directory for processing; provide sufficient capacity for the expected packages.
Mount any additional writable paths enabled by your configuration.
`/home/nonroot` is the default home and working directory; with a read-only root
filesystem, add a writable mount there only if your configuration needs it. A
deployment can override the user, provided its mounted files and directories
have matching permissions.

Database migrations are embedded and run at startup by default. Follow the
[upgrade procedure] before upgrading an existing deployment.

## Build and check locally

Use GoReleaser 2.19.0 (pinned in `.bine.toml`), the Go version in `go.mod`,
Node.js 24.22.1, and Docker with Buildx and the containerd image store enabled.
CI uses Docker 29.5.2. The containerd image store makes `docker image save`
include the OCI layout that Dagger imports; the export helper rejects archives
without it. Install Dagger 1.0.0-beta.24, matching
`hack/dagger/dagger-module.toml`, to run the service tests.

Run from the repository root:

```sh
make container-build
make container-export
make container-check CONTAINER_ARCH=arm64 CONTAINER_ARTIFACTS=dist/containers
make container-startup
```

Choose `amd64` or `arm64` for `container-check` to match your native platform.
Without `CONTAINER_ARCH`, that target checks both architectures and requires
emulation for the foreign architecture. `container-startup` defaults to the
architecture reported by `go env GOARCH`; override `CONTAINER_ARCH` if needed.
It exports results to `hack/dagger/runtime/startup-artifacts`.

`container-build` runs `goreleaser release --clean --snapshot`. It builds the
frontend, both binaries, the existing archive, and both production container
images without publishing. GoReleaser loads snapshot images into the local
Docker daemon with `-amd64` and `-arm64` tag suffixes. Do not add
`--skip=publish`: GoReleaser's container build is part of its publish phase, and
that flag would skip the images too. The Dockerfile consumes GoReleaser's build
context and is not intended for a standalone `docker build .`.

`container-export` saves those existing images without rebuilding. It writes
`dist/containers/enduro-amd64.tar`, `dist/containers/enduro-arm64.tar`, and
`dist/containers/metadata.json`. The metadata records the version, source
commit, image tags and IDs, and archive checksums. `container-check` verifies
the archive checksum and loaded image identity before checking architecture,
labels, nonroot user, entrypoint, default configuration path, home, and working
directory. It runs `--version` with networking disabled, a read-only filesystem,
dropped capabilities, and no new privileges, and checks the
missing-configuration failure. Omit `CONTAINER_ARTIFACTS` to check the already
loaded snapshot images.

`container-startup` imports the native production archive into Dagger and starts
MySQL and Temporal to check migrations, the API, embedded dashboard assets, and
service shutdown. It does not start Archivematica. The [smoke harness] documents
the full Archivematica and object-storage scenarios and the separate source
coverage mode.

## CI and releases

CI builds the frontend once and passes its public assets to the snapshot build
and the separate source coverage build. GoReleaser builds the production images
once; test jobs receive the exported archives without registry credentials.
Native AMD64 and ARM64 jobs check image identity and run startup acceptance.
Pull requests and pushes to `main` also run the full Archivematica, SeaweedFS,
and argmin suites on AMD64 against the production archive, plus a separate
source-built smoke suite that collects Go coverage using the same production
Dockerfile. Production image tests do not rebuild Enduro or collect coverage.

Pushes to `dev/containers` use a shorter branch-iteration mode: native startup
acceptance on both architectures and Dagger's `coverage-check` instead of the
full service suites. The coverage check builds a real instrumented binary, runs
`--version`, and verifies that Go coverage data can be exported without starting
services. A cold compile can still take several minutes. This mode does not
provide the integration coverage of the full suites. To run those suites on the
branch before final review, manually dispatch the Test workflow with
`full-tests` enabled (the default). Pull requests, `main`, and releases use full
acceptance.

A tagged release first builds a candidate multi-platform image and a draft
GitHub release. The reusable container tests pull the candidate by immutable
index digest and run the same native AMD64 and ARM64 startup and full AMD64
integration checks. After they pass, the workflow promotes the whole tested
index to the exact Git tag without rebuilding, publishes the draft release, and
closes the matching milestone. Failed tests leave the candidate and draft
available for inspection without promoting the release tag.

To retry failed test or promotion jobs, use GitHub Actions' **Re-run failed
jobs** so they reuse the existing candidate. A full rebuild is refused when a
draft or published release already exists for the tag, keeping the release
archives and container image together. If the candidate job itself failed after
creating a draft, inspect and remove that incomplete draft before rebuilding.
Never remove a draft whose image was already promoted; retry its failed
downstream jobs instead. Promotion never overwrites a final tag with a different
digest.

The GitHub Actions token needs permission to write packages; an existing GHCR
package must grant this repository access. Package visibility is managed
separately in GitHub. Container SBOM reporting is separate follow-up work and is
not implemented by this workflow.

[configuration reference]: ./configuration-reference.md
[security guide]: ./security-configuration.md
[smoke harness]: ../hack/dagger/smoke/README.md
[transfer publisher]: ./configuration-reference.md#pipelinetransferpublisher
[upgrade procedure]: ./upgrades.md#upgrade-enduro
