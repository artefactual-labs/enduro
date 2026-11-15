# Refresh dependencies

Use this guide to update dependencies in coherent groups, with their related
pins, generated output, and validation. Run commands from the repository root
unless a subshell changes directories. Deployment upgrades have a separate
[upgrade procedure].

Runtime dependencies include Go libraries, browser assets, the production image
base, and external service contracts. Workflow dependencies include generators,
build/test tools, Dagger, and CI. Some manifests contain both: the root Go
module also owns helpers, tests, and bine; the frontend includes both browser
libraries and build tools. Production serves generated static frontend assets
without requiring Node or the complete npm graph.

## Working sequence

1. Establish scope and baseline. Inspect existing changes, recent dependency
   commits and releases, and the current manifests. Use recent check results for
   unchanged inputs or run the relevant checks before editing to distinguish
   existing failures from regressions.
2. Select one update group from the sections below. Verify target versions
   against upstream metadata and read the release notes between the current and
   intended versions, including compatibility requirements and breaking changes.
   Prefer compatible updates; review major migrations and prerelease channels
   deliberately.
3. Update that group's coupled pins and configuration. Include transitive
   dependencies and overrides in the review. A related component can stay pinned
   when compatible; explain the reason.
4. When a generator, its inputs, or its API changes, regenerate derived files
   with the owning commands. A generator update requires regeneration even with
   unchanged inputs. Review warnings and output, and keep the pins with their
   regenerated code and metadata.
5. Run focused checks after each group. Investigate new failures while the
   responsible changes are still easy to identify. If an upgrade cannot be
   completed, defer that upgrade and report the limitation.
6. Once the selected pins are final, run the shared integration and packaging
   checks for the affected behavior. Summarize completed changes, deferred
   updates, remaining advisories, and checks that passed or were not run.

For example, start with:

```sh
git status --short
git diff --stat
git diff --cached --stat
git log -15 --oneline
gh release list --limit 4
```

Preserve existing work and staged changes. Keep authoritative versions in
manifests instead of copying them into this guide. The [reference inventory]
helps locate less frequent update targets; it is not a requirement to update
unrelated utilities, fixtures, or editor settings during every pass.

### Verify the effective toolchains

Check the actual compiler in each module being validated:

```sh
go version
(cd hack/dagger && go version)
```

The `go` and `toolchain` directives are not exact execution pins. A newer
installed compiler can be selected, and Dagger's module build/runtime and CI can
use different compilers from the host. Identify those effective versions
separately when changing Go or interpreting standard-library vulnerabilities. A
scan forced onto a particular toolchain establishes results for that toolchain;
confirm the compiler used by the relevant build before dismissing a finding. See
[Go toolchain selection].

Keep the application Go requirement aligned with `goImage` in the harness. Check
Dagger's own generator/compiler compatibility before changing its module
requirement; matching the application module is not sufficient. Align
`nodeImage` with Node installation in test/release workflows, and validate the
frontend with that Node version.

## Update groups

### Root Go module and internal helpers

The root `go.mod` and `go.sum` cover application libraries, Go test
dependencies, internal helpers, and the bine `tool` dependency. Review all
relevant parts of this graph:

```sh
go list -m -u all
```

Select reviewed versions with `go get module/path@version`, then run
`go mod tidy`. `make deps-upgrade-direct` is available for a broader direct
dependency update. `go get -u ./...` also refreshes dependencies of imported
packages; review the resulting scope rather than updating every unused module in
the graph.

Bine's module version and the binary versions in `.bine.toml` are separate
update targets. Use the [Go checks] below, including helper tests excluded by
`make test`. Review copied third-party source separately: Go module tools do not
update `internal/pipeline/sync/semaphore/`. Preserve its local behavior and
accompanying license when incorporating relevant upstream fixes.

### Temporal libraries, CLI, and services

Treat Temporal as a compatibility set across runtime and workflow resources:

- Root `go.mod`: `go.temporal.io/api`, `go.temporal.io/sdk`,
  `go.temporal.io/sdk/contrib/opentelemetry`, and the shared
  `github.com/artefactual-sdps/temporal-activities` dependency.
- `.bine.toml`: the host `temporal-cli` used by `make temporal`.
- `docker-compose.yml` and `hack/dagger/main.go`: server and admin-tools images,
  plus the Compose UI image. The harness CLI comes from the admin-tools image,
  independently of the bine pin; `hack/tctl.sh` uses that image too.

Read the [SDK], [API], [CLI], and [server release notes]. Check the SDK's API
requirements, telemetry compatibility, Go requirements, deprecations, workflow
determinism, and payload/serialization behavior. These components have
independent version numbers; choose a compatible set without forcing equal
numbers or changing every component on every pass.

Keep server/admin-tools images aligned across Compose and Dagger. Review the UI
independently. Check schema setup, namespace creation, dynamic configuration,
dev-server flags, and workflow-history export commands when their owning
component changes.

Run the Go checks for library updates. Exercise the development-server path for
host CLI updates and the server/admin-tools schema path for service image or
setup changes. Use the relevant [integration checks] for affected workflow
submission, status/history, operator decisions, batch processing, and trace
capture. Deployment still follows the [upgrade procedure] for durable workflows
and persistent schemas.

### Frontend libraries and build/test tooling

Use the [frontend package workflow] to select updates and refresh
`frontend/package.json` with its lockfile. Review Nuxt, Vue/router, UI and icon
packages alongside build, lint, typecheck, test, and coverage dependencies.

Select the intended Nuxt release/channel **before** invoking
`npm run upgrade:nuxt --prefix frontend`. Its configured `nuxt upgrade --dedupe`
command installs directly and does not offer target approval. Pass an explicit
reviewed channel when needed, or install a reviewed package version directly.
Then use the existing interactive dependency scripts for remaining updates.
After selecting dependencies, run `npm install --prefix frontend` to synchronize
the lockfile before validation; `npm ci` does not update it.

Review `overrides` separately and preserve their reasons unless the constraint
has been resolved. Review Nuxt compatibility settings, experimental flags, Vue
deduplication, and upstream workarounds with framework/router changes. Do not
advance a compatibility date or remove a workaround without checking the
behavior it protects. Inspect preparation changes to `frontend/.nuxtrc`.

Run the [frontend checks]. Do not force audit fixes that introduce unrelated
major migrations or framework downgrades.

### Goa, OpenAPI Generator, and mocks

For Goa, coordinate these resources:

- `goa.design/goa/v3` and `goa.design/plugins/v3` in `go.mod`.
- The `goa` generator in `.bine.toml`; align it with the Goa module and select
  compatible plugins.
- Generated Go bindings and schemas under `internal/api/gen/`, then the
  generated frontend client/runtime and metadata.

Read the [Goa release notes] and [plugin tag history] before editing pins. Check
Go requirements, generator/runtime compatibility, DSL/schema changes,
validation, serialization, and transport behavior. Compare plugin tags when
separate release notes are unavailable.

Regenerate in dependency order, even with an unchanged API design:

```sh
make gen-goa
make frontend-client
```

An **OpenAPI Generator image change alone** in `Makefile` also requires
`make frontend-client`, even when the API specification is unchanged. This
target applies `frontend/scripts/patch-openapi-runtime.mjs`; do not patch the
generated runtime by hand. Use `make gen-mock` when mockgen or relevant
interfaces change.

Review warnings, generated Go code, schemas, client/runtime code, and metadata.
Confirm intended request/response changes, optional fields, defaults, enums, and
errors. Run Go checks for changed Go libraries, bindings, or mocks, and frontend
checks for changed clients or runtimes, plus affected API/browser scenarios.
Include the generator/module pins and all derived output in the same change.

### Dagger, browser tests, and test images

Dagger has a separate Go module. Review `dagger.toml`, the external Go SDK and
helper resolutions in `dagger.lock`, `hack/dagger/dagger-module.toml`, and
`hack/dagger/go.mod`/`go.sum`. For engine/SDK updates, align the CLI, module
engine version, and Dagger installation in CI, then refresh and regenerate:

```sh
dagger lock update
dagger generate --dagger-go-sdk -y --require-load
```

Regenerate after module API changes too. For image, source-revision, and
Playwright pin updates, update the applicable locks and run the affected checks;
regenerate the SDK only if its inputs or the module API also changed.

Inspect the entire generated diff. The SDK currently rewrites logging
replacements to older defaults: restore the compatible replacements documented
in `hack/dagger/go.mod` before validating. Preserve its related OpenTelemetry
and grpc-gateway constraints until upstream support changes. Recheck those
comments and the upstream issue each time. Avoid blanket upgrades of this
module's transitive graph.

Keep the exact Playwright package in `hack/dagger/smoke/package.json` aligned
with `playwrightImage` in the harness. Update its lockfile and use the
[smoke harness] for execution and diagnostics.

Review every image reference affected by a base/service update, including inline
Alpine/Debian references. Keep shared MySQL, Temporal, Redis, SeaweedFS, and
LGTM versions aligned across the harness and Compose. Review `apt-get`/`apk`
utilities when their base changes. A base digest does not pin subsequently
downloaded OS packages.

For ambox, review the bundled Archivematica/Storage Service changes and validate
the integration. For argmin, review its source revision, upstream Cargo
lockfile, Rust build image, and Debian runtime together; retain
`cargo build --locked`. MinIO/Coollabs and Silo remain separately pinned
compatibility targets.

Commit regenerated bindings and lockfiles together. `dagger lock update`
refreshes recorded entries; new image references are recorded when resolved. See
[image pin maintenance] for a resolution-only recipe. Availability alone does
not establish compatibility: run the harness module and integration checks.

### Production images, CI actions, and development binaries

For production-base or packaging changes, review `Dockerfile` and
`.goreleaser.yml` together: certificates, user/permissions, runtime environment,
architecture support, and archive/image contents. Use the production-image
checks in the [container guide].

Review actions in `.github/workflows/*.yml` and all relevant binaries in
`.bine.toml`. Keep GoReleaser aligned with test/release workflows and
gh-actions-lock/jactionlint aligned with their workflow installation/action
references. Review runner OS/architecture, Docker Engine/Buildx inputs, and
action-installed tool versions/defaults. An action source pin does not
necessarily pin every binary it downloads.

After workflow dependencies change:

```sh
make lock-actions
make check-actions
make lint-actions
```

Review `.github/workflows/actions.lock`, including direct/transitive commits and
repository identities. Do not hand-edit generated lockfiles. Check tool
configuration when its owner changes, such as `.golangci.yml`,
`.github/jactionlint.yaml`, `.codecov.yml`, or frontend lint/test settings. For
local authentication or tool lookup failures, see [tool troubleshooting].

## Validation

Compare results with the baseline. Run focused checks as each group changes and
shared integration checks against the final selections. Reuse passing results
while their inputs and toolchains remain unchanged; repeat checks when later
changes or unresolved failures require them. A full refresh normally needs all
applicable groups below; a narrow update needs its own checks and any affected
integrations.

Refresh vulnerability checks against current advisories. Run `npm audit` in each
affected npm project and review its dependency paths and proposed fixes.

### Go checks

For root-module changes:

```sh
go mod tidy -diff
make test
make lint LINT_FLAGS=--fix=0
go test ./hack/s3put ./hack/s3setup ./hack/gencols
```

The explicit helper command covers tests excluded by `make test` and compiles
`gencols`. For harness-module changes, also run:

```sh
(cd hack/dagger && go mod tidy -diff && go test ./...)
```

Use the nightly govulncheck workflow as the reference for application scanning.
Scan the Dagger module separately and identify the effective compiler for each
build being assessed. Separate existing advisories, new findings, and documented
upstream constraints; changing the scan toolchain alone does not establish that
the actual build is fixed.

### Frontend checks

For changes to frontend dependencies, application code, build/test
configuration, or the generated client:

```sh
npm ci --prefix frontend
npm run lint --prefix frontend
npm run typecheck --prefix frontend
npm run test:coverage --prefix frontend
npm run build --prefix frontend
```

### Smoke tooling checks

For changes to `hack/dagger/smoke` dependencies or configuration, verify
installation and discovery, then run the affected smoke scenarios:

```sh
npm ci --prefix hack/dagger/smoke
npm test --prefix hack/dagger/smoke -- --list
```

### Integration and packaging checks

For source-built harness runs, reuse frontend assets with
`--frontend-assets frontend/.output/public` when they were built from unchanged
frontend inputs with the selected Node/toolchain versions. Otherwise, build the
frontend first. The [smoke harness] documents source/image modes and artifact
handling.

- For changed Compose definitions, run `docker compose config --quiet`. Include
  `docker-compose.observability.yml` when its services change.
- For GoReleaser changes, run `go tool bine run goreleaser check`.
- Run startup acceptance when the runtime image, startup configuration, or
  migration behavior changes. Run the smoke scenarios affected by application,
  service, Dagger, or browser-tool changes. Use `hack/dagger/run.sh` for these
  checks.
- For storage changes, run the affected providers or
  `make test-smoke-object-storage`. Run harness suites sequentially because they
  share runtime cache volumes.
- For production-base or packaging changes, run the archive and native image
  checks in the [container guide]. Source-built smoke tests do not replace
  production-image validation.
- Before handing off, run `git diff --check`. Report omitted checks and their
  practical limits, including any architecture or provider not exercised.

## Reference inventory

Use this list to find update locations and related compatibility surfaces.
Review entries relevant to the selected scope; review the full dependency
surface for a broad refresh. Update this list when a new dependency-bearing
resource appears. It does not turn every configuration or fixture into a
separate versioned dependency.

### Runtime resources

- Go libraries/standard library: root `go.mod`, `go.sum`, application imports.
  Copied source: `internal/pipeline/sync/semaphore/` and
  `internal/pipeline/sync/LICENSE`.
- Browser libraries, icons, generated client, and shipped assets:
  `frontend/package.json`, `frontend/package-lock.json`, `frontend/app/`,
  `frontend/public/`, and `frontend/frontend.go`.
- Production image and release contents: `Dockerfile`, `.goreleaser.yml`.
- External services: MySQL, Temporal, optional S3 providers/Redis,
  Archivematica, Storage Service, SFTP, production-system/Hari integrations,
  reverse proxies, metrics and OTLP endpoints. Review `enduro.toml`,
  `internal/pipeline/`, `internal/publisher/`, `internal/nha/`, and
  installation/configuration docs. Production installations are managed
  separately from repository test images.

### Workflow resources and compatibility surfaces

- Tools: root Go `tool`/test graph, every `.bine.toml` entry, and `Makefile`.
  This includes bine, golangci-lint, GoReleaser, go-mod-outdated, gotestsum,
  Goa, mockgen, Temporal CLI, jq, gh-actions-lock, and jactionlint.
- Frontend tooling/settings: npm manifests, `frontend/nuxt.config.ts`,
  `frontend/.nuxtrc`, `frontend/eslint.config.mjs`, `frontend/tsconfig.json`,
  `frontend/vitest.config.ts`, and `frontend/vitest.setup.nuxt.ts`.
- Generators and output: `internal/api/design/`, `internal/api/gen/`,
  `frontend/app/openapi-generator/`, its metadata,
  `frontend/scripts/patch-openapi-runtime.mjs`, and generated mocks.
- Dagger/Playwright: `dagger.toml`, `dagger.lock`,
  `hack/dagger/dagger-module.toml`, harness Go manifests/bindings, and
  `hack/dagger/smoke/` npm manifests, configuration, tests, and startup script.
- Test/build images: `hack/dagger/main.go`, both Compose files, and the Dagger
  lock. Check MySQL, Temporal server/admin-tools/UI, Redis, SeaweedFS,
  MinIO/Coollabs, Silo, ambox, LGTM, Go, Node, Rust, Playwright, Alpine, and
  Debian; argmin's source revision/upstream Cargo lock; and image-installed OS
  utilities.
- CI/update automation: `.github/workflows/*.yml`,
  `.github/workflows/actions.lock`, `.github/dependabot.yml`,
  `.github/jactionlint.yaml`, and `frontend/renovate.json`.
- Quality/editor/release settings: `.golangci.yml`, `.codecov.yml`,
  `.goreleaser.yml`, `.editorconfig`, `.markdownlint.json`,
  `.vscode/settings.json`, `.gitattributes`, and harness attributes.
- Internal utilities: `hack/s3put/`, `hack/s3setup/`, `hack/gencols/`,
  `hack/test-dpj/`, `hack/tctl.sh`, `hack/containers/*.sh`,
  `hack/dagger/run.sh`, and `hack/dagger/test-failure-artifacts.sh`.
- Service setup/configuration: `hack/compose/*.sh`,
  `hack/docker-init-mysql.sql`,
  `hack/etc/temporal/dynamicconfig/development-sql.yaml`,
  `hack/seaweedfs/notification.toml`, and `hack/etc/prometheus.yaml`.
- Host/documentation utilities: prerequisites in Makefiles, scripts and READMEs,
  including Git/GitHub CLI, Go, Node/npm, Docker/Compose/Buildx, Make, shells,
  archive/checksum tools, curl/jq, mysql CLI, mc, rsync, uuidgen, shuf, and
  uv/uvx/mdformat plugins. Review availability and flags when their workflow
  changes. Run fixture loaders/uploaders only with disposable test data.
- Committed fixtures, schemas, assets, licenses, helper READMEs, and operator
  guidance: maintain them when behavior or instructions change. Ordinary URLs,
  SVG namespaces, screenshots, and fixture format labels are not package pins.

[Dependabot] covers both Go modules, both npm projects, Actions, and the root
Dockerfile; it does not cover every tool, inline image, source revision, or
configuration. The frontend's [Renovate configuration] alone does not establish
that Renovate is active. Supplement manifest review with searches of the harness
image/source references and install/download commands when scoping a full
refresh.

Verify available versions through Go module and npm registry metadata, published
upstream tags, and image registry digests as well as release notes. Release
listings can omit prereleases or tags without a release entry. Compare
candidates with the selected release line and current pin: a default `latest`
result can represent a major migration or be older than an intentional
prerelease or pseudo-version. Check mutable image digests even when their tags
are unchanged.

Manifests and lock/checksum files record dependency selections and integrity,
not every build input. `go.sum` records checksums; module selection comes from
`go.mod` and module metadata. OS repositories, runners, action-installed
binaries, and unpinned update tools can resolve independently. Record the actual
tool versions and image/source digests needed to reproduce the refresh.

[api]: https://github.com/temporalio/api-go/releases
[cli]: https://github.com/temporalio/cli/releases
[container guide]: ./containers.md
[dependabot]: ../.github/dependabot.yml
[frontend checks]: #frontend-checks
[frontend package workflow]: ../frontend/README.md#package-management
[go checks]: #go-checks
[go toolchain selection]: https://go.dev/doc/toolchain#select
[goa release notes]: https://github.com/goadesign/goa/releases
[image pin maintenance]: ../hack/dagger/smoke/README.md#refresh-image-pins
[integration checks]: #integration-and-packaging-checks
[plugin tag history]: https://github.com/goadesign/plugins/tags
[reference inventory]: #reference-inventory
[renovate configuration]: ../frontend/renovate.json
[sdk]: https://github.com/temporalio/sdk-go/releases
[server release notes]: https://github.com/temporalio/temporal/releases
[smoke harness]: ../hack/dagger/smoke/README.md
[tool troubleshooting]: ./contributing.md#tool-lookup-and-authentication
[upgrade procedure]: ./upgrades.md
