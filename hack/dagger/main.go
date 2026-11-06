// Enduro integration-test harnesses.
package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dagger/enduro-e-2-e/internal/dagger"
)

const (
	mysqlImage        = "mysql:8.4.13-oraclelinux9"
	temporalImage     = "temporalio/server:1.32.1"
	temporalToolImage = "temporalio/admin-tools:1.32.1"
	goImage           = "golang:1.27.3-bookworm"
	nodeImage         = "node:24.22.1-bookworm"
	redisImage        = "redis:8.2.10-alpine3.22"
	minioLatestImage  = "quay.io/minio/minio:RELEASE.2025-09-07T16-13-09Z"
	argminBuildImage  = "rust:1.99.1-bookworm"
	argminRevision    = "8ebbf735460e1e6e38333571853b460c40d09879"
	seaweedFSImage    = "chrislusf/seaweedfs:4.48"
	amboxImage        = "ghcr.io/sevein/ambox:latest"
	playwrightImage   = "mcr.microsoft.com/playwright:v1.64.1-noble"
	lgtmImage         = "grafana/otel-lgtm:0.38.0"
)

type EnduroE2E struct {
	// +private
	ReportOnly bool
}

// Export smoke diagnostics even when Playwright fails. Callers using reportOnly
// must check the exported smoke-exit-code files, as hack/dagger/run.sh does.
func New(
	// +optional
	reportOnly bool,
) *EnduroE2E {
	return &EnduroE2E{ReportOnly: reportOnly}
}

type objectStorageProvider string

const (
	objectStorageProviderMinioLatest objectStorageProvider = "minio-latest"
	objectStorageProviderSeaweedFS   objectStorageProvider = "seaweedfs"
	objectStorageProviderArgmin      objectStorageProvider = "argmin"
)

type runtimeVolumes struct {
	batch      *dagger.CacheVolume
	watched    *dagger.CacheVolume
	completed  *dagger.CacheVolume
	transfers  *dagger.CacheVolume
	processing *dagger.CacheVolume
	receipts   *dagger.CacheVolume
	storage    *dagger.CacheVolume
	coverage   *dagger.CacheVolume
}

type smokeEnvironment struct {
	coverage    bool
	cacheBuster string
	source      *dagger.Directory
	volumes     runtimeVolumes
	mysql       *dagger.Service
	temporal    *dagger.Service
	ambox       *dagger.Service
	enduro      *dagger.Service
	redis       *dagger.Service
	storage     *dagger.Service
	lgtm        *dagger.Service
}

// Run the smoke test suite against ambox.
//
// This starts Enduro, Temporal, MySQL, and an ambox Archivematica environment,
// then submits a zipped transfer through Enduro's filesystem watcher. Enduro
// prepares the transfer locally and publishes it to ambox over SFTP, which
// exercises the publish, transfer, cleanup, and completion workflow path.
//
// It also drives the Nuxt batch-import form with Playwright, submitting a
// simple directory transfer from a filesystem path visible to Enduro. The suite
// waits for processing to finish, downloads the generated AIPs through Enduro,
// and performs basic checks on the AIPs and Temporal history. It returns a
// Dagger directory with downloaded AIPs and diagnostic reports for local
// inspection or CI artifact upload.
//
// Call from the repository root with:
//
//	dagger -m hack/dagger api call smoke-tests --source . export --path hack/dagger/runtime/artifacts
//
// +cache="session"
func (m *EnduroE2E) SmokeTests(ctx context.Context,
	// +ignore=[".git", "build", "dist", "**/node_modules", "frontend/.nuxt", "frontend/.output", "frontend/coverage", "frontend/.cache", ".ruff_cache", ".playwright-cli", "covreport", "enduro", "hack/dagger/runtime", "hack/dagger/internal", "hack/dagger/dagger.gen.go", "dagger.lock", "hack/minio-data", "hack/seaweedfs-data"]
	source *dagger.Directory,
	// Production image in an OCI-compatible archive; mutually exclusive with imageRef.
	// +optional
	imageArchive *dagger.File,
	// Production registry image pinned by sha256 digest.
	// +optional
	imageRef string,
	// Reuse built frontend assets for the default source coverage build.
	// +optional
	frontendAssets *dagger.Directory,
) (*dagger.Directory, error) {
	image, coverage, err := m.runtimeImage(ctx, source, imageArchive, imageRef, frontendAssets)
	if err != nil {
		return nil, err
	}
	env, err := m.smokeEnvironment(ctx, source, image, coverage)
	if err != nil {
		return nil, err
	}

	return m.runSmokeSuite(ctx, env)
}

// Run S3 watcher smoke tests against MinIO, SeaweedFS, and argmin.
//
// The MinIO scenario exercises native Redis notifications configured through
// environment variables and the S3 API. The SeaweedFS scenario exercises
// SeaweedFS filer webhooks, Enduro's object event webhook,
// normalized Redis events, and the S3 watcher path. All scenarios publish the
// transfer to ambox and verify the resulting AIP. Argmin uses a synthetic
// normalized Redis event after upload because it has no native notifications.
//
// Call from the repository root with:
//
//	dagger -m hack/dagger api call object-storage-smoke-tests --source . export --path hack/dagger/runtime/object-storage-artifacts
//
// +cache="session"
func (m *EnduroE2E) ObjectStorageSmokeTests(ctx context.Context,
	// +ignore=[".git", "build", "dist", "**/node_modules", "frontend/.nuxt", "frontend/.output", "frontend/coverage", "frontend/.cache", ".ruff_cache", ".playwright-cli", "covreport", "enduro", "hack/dagger/runtime", "hack/dagger/internal", "hack/dagger/dagger.gen.go", "dagger.lock", "hack/minio-data", "hack/seaweedfs-data"]
	source *dagger.Directory,
	// Production image in an OCI-compatible archive; mutually exclusive with imageRef.
	// +optional
	imageArchive *dagger.File,
	// Production registry image pinned by sha256 digest.
	// +optional
	imageRef string,
	// Reuse built frontend assets for the default source coverage build.
	// +optional
	frontendAssets *dagger.Directory,
) (*dagger.Directory, error) {
	image, coverage, err := m.runtimeImage(ctx, source, imageArchive, imageRef, frontendAssets)
	if err != nil {
		return nil, err
	}
	minioLatestArtifacts, err := m.objectStorageSmokeTest(ctx, source, image, coverage, objectStorageProviderMinioLatest)
	if err != nil {
		return nil, err
	}

	seaweedFSArtifacts, err := m.objectStorageSmokeTest(ctx, source, image, coverage, objectStorageProviderSeaweedFS)
	if err != nil {
		return nil, err
	}

	argminArtifacts, err := m.objectStorageSmokeTest(ctx, source, image, coverage, objectStorageProviderArgmin)
	if err != nil {
		return nil, err
	}

	return dag.Directory().
		WithDirectory("argmin", argminArtifacts).
		WithDirectory(string(objectStorageProviderMinioLatest), minioLatestArtifacts).
		WithDirectory("seaweedfs", seaweedFSArtifacts), nil
}

// Exercise the coverage build and production runtime without external services.
// This checks that Enduro starts as nonroot, prints its version, and writes Go
// coverage data. It does not exercise ingestion or replace the full smoke suite.
func (m *EnduroE2E) CoverageCheck(ctx context.Context,
	// +ignore=[".git", "build", "dist", "**/node_modules", "frontend/.nuxt", "frontend/.output", "frontend/coverage", "frontend/.cache", ".ruff_cache", ".playwright-cli", "covreport", "enduro", "hack/dagger/runtime", "hack/dagger/internal", "hack/dagger/dagger.gen.go", "dagger.lock", "hack/minio-data", "hack/seaweedfs-data"]
	source *dagger.Directory,
	// Reuse built frontend assets for the source coverage build.
	// +optional
	frontendAssets *dagger.Directory,
) (*dagger.Directory, error) {
	image, _, err := m.runtimeImage(ctx, source, nil, "", frontendAssets)
	if err != nil {
		return nil, err
	}
	user, err := image.User(ctx)
	if err != nil {
		return nil, err
	}
	if user != "65532:65532" {
		return nil, fmt.Errorf("expected image user 65532:65532, got %q", user)
	}
	checked := image.
		WithDirectory("/coverage", dag.Directory(), dagger.ContainerWithDirectoryOpts{Owner: "65532:65532"}).
		WithEnvVariable("GOCOVERDIR", "/coverage").
		WithExec([]string{"--version"}, dagger.ContainerWithExecOpts{UseEntrypoint: true})
	output, err := checked.Stdout(ctx)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(output, "enduro version ") {
		return nil, fmt.Errorf("unexpected version output: %q", output)
	}
	converter := dag.Container().
		From(goImage).
		WithDirectory("/coverage", checked.Directory("/coverage")).
		WithNewFile("/artifacts/version.txt", output).
		WithExec([]string{"sh", "-ceu", strings.TrimSpace(`
			test -n "$(find /coverage -type f -name 'covcounters.*' -print -quit)"
			go tool covdata textfmt -i=/coverage -o=/artifacts/enduro-check.coverprofile
			go tool covdata percent -i=/coverage > /artifacts/enduro-check-coverage.txt
		`)})
	if _, err := converter.Stdout(ctx); err != nil {
		return nil, err
	}
	return converter.Directory("/artifacts"), nil
}

// Check a native-platform image's configuration, database migrations, API,
// embedded dashboard assets, and service shutdown without Archivematica.
//
// +cache="session"
func (m *EnduroE2E) StartupTest(ctx context.Context,
	// +ignore=[".git", "build", "dist", "**/node_modules", "frontend/.nuxt", "frontend/.output", "frontend/coverage", "frontend/.cache", ".ruff_cache", ".playwright-cli", "covreport", "enduro", "hack/dagger/runtime", "hack/dagger/internal", "hack/dagger/dagger.gen.go", "dagger.lock", "hack/minio-data", "hack/seaweedfs-data"]
	source *dagger.Directory,
	// Production image in an OCI-compatible archive; mutually exclusive with imageRef.
	// +optional
	imageArchive *dagger.File,
	// Production registry image pinned by sha256 digest.
	// +optional
	imageRef string,
	// Reuse built frontend assets for the default source coverage build.
	// +optional
	frontendAssets *dagger.Directory,
) (*dagger.Directory, error) {
	image, coverage, err := m.runtimeImage(ctx, source, imageArchive, imageRef, frontendAssets)
	if err != nil {
		return nil, err
	}
	user, err := image.User(ctx)
	if err != nil {
		return nil, err
	}
	if user != "65532:65532" {
		return nil, fmt.Errorf("expected image user 65532:65532, got %q", user)
	}
	entrypoint, err := image.Entrypoint(ctx)
	if err != nil {
		return nil, err
	}
	if len(entrypoint) != 1 || entrypoint[0] != "/usr/local/bin/enduro" {
		return nil, fmt.Errorf("expected direct Enduro entrypoint, got %q", entrypoint)
	}
	cacheBuster := fmt.Sprintf("startup-%d", time.Now().UnixNano())
	mysql := m.mysqlService(source)
	mysql, err = mysql.Start(ctx)
	if err != nil {
		return nil, err
	}
	temporal := m.temporalDevService()
	if err := m.waitTemporalNamespace(ctx, temporal, cacheBuster); err != nil {
		return nil, err
	}
	runtime := image.WithServiceBinding("mysql", mysql).
		WithServiceBinding("temporal", temporal).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithDirectory("/runtime/watched", dag.Directory(), dagger.ContainerWithDirectoryOpts{Owner: "65532:65532"}).
		WithNewFile("/etc/enduro/enduro.toml", `
verbosity = 2
[api]
listen = "0.0.0.0:9000"
[database]
dsn = "enduro:enduro123@tcp(mysql:3306)/enduro"
[temporal]
namespace = "default"
address = "temporal:7233"
taskQueue = "global"
[[watcher.filesystem]]
name = "startup"
path = "/runtime/watched"
inotify = true
`).WithExposedPort(9000)
	// Startup coverage isn't aggregated; the full suites collect workflow coverage.
	if coverage {
		runtime = runtime.WithEnvVariable("GOCOVERDIR", "/tmp")
	}
	service := runtime.AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
	tester := dag.Container().From(nodeImage).WithServiceBinding("enduro", service).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithFile("/startup.mjs", source.File("hack/dagger/smoke/startup.mjs")).
		WithExec([]string{"node", "/startup.mjs"})
	if _, err := tester.Stdout(ctx); err != nil {
		return nil, err
	}
	if _, err := service.Stop(ctx); err != nil {
		return nil, err
	}
	return tester.Directory("/artifacts").WithNewFile("runtime.txt", "user="+user+"\nentrypoint="+entrypoint[0]+"\nstop=completed\n"), nil
}

// Run one S3 watcher smoke test provider.
//
// Provider must be "minio-latest", "seaweedfs", or "argmin".
//
// +cache="session"
func (m *EnduroE2E) ObjectStorageSmokeTest(ctx context.Context,
	// +ignore=[".git", "build", "dist", "**/node_modules", "frontend/.nuxt", "frontend/.output", "frontend/coverage", "frontend/.cache", ".ruff_cache", ".playwright-cli", "covreport", "enduro", "hack/dagger/runtime", "hack/dagger/internal", "hack/dagger/dagger.gen.go", "dagger.lock", "hack/minio-data", "hack/seaweedfs-data"]
	source *dagger.Directory, provider string,
	// Production image in an OCI-compatible archive; mutually exclusive with imageRef.
	// +optional
	imageArchive *dagger.File,
	// Production registry image pinned by sha256 digest.
	// +optional
	imageRef string,
	// Reuse built frontend assets for the default source coverage build.
	// +optional
	frontendAssets *dagger.Directory,
) (*dagger.Directory, error) {
	image, coverage, err := m.runtimeImage(ctx, source, imageArchive, imageRef, frontendAssets)
	if err != nil {
		return nil, err
	}
	switch objectStorageProvider(provider) {
	case objectStorageProviderMinioLatest:
		return m.objectStorageSmokeTest(ctx, source, image, coverage, objectStorageProviderMinioLatest)
	case objectStorageProviderArgmin:
		return m.objectStorageSmokeTest(ctx, source, image, coverage, objectStorageProviderArgmin)
	case objectStorageProviderSeaweedFS:
		return m.objectStorageSmokeTest(ctx, source, image, coverage, objectStorageProviderSeaweedFS)
	default:
		return nil, fmt.Errorf("unknown object storage provider %q", provider)
	}
}

func (m *EnduroE2E) objectStorageSmokeTest(ctx context.Context, source *dagger.Directory, image *dagger.Container, coverage bool, provider objectStorageProvider) (*dagger.Directory, error) {
	env, err := m.objectStorageEnvironment(ctx, source, image, coverage, provider)
	if err != nil {
		return nil, err
	}

	return m.runObjectStorageSmokeSuite(ctx, env, provider)
}

func (m *EnduroE2E) smokeEnvironment(ctx context.Context, source *dagger.Directory, image *dagger.Container, coverage bool) (*smokeEnvironment, error) {
	cacheBuster := fmt.Sprintf("%d", time.Now().UnixNano())
	volumes := runtimeVolumes{
		batch:      dag.CacheVolume("enduro-e2e-ambox-batch"),
		watched:    dag.CacheVolume("enduro-e2e-ambox-watched"),
		completed:  dag.CacheVolume("enduro-e2e-ambox-completed"),
		transfers:  dag.CacheVolume("enduro-e2e-ambox-transfers"),
		processing: dag.CacheVolume("enduro-e2e-ambox-processing"),
		receipts:   dag.CacheVolume("enduro-e2e-ambox-receipts"),
		storage:    dag.CacheVolume("enduro-e2e-ambox-storage"),
		coverage:   dag.CacheVolume("enduro-e2e-ambox-coverage"),
	}

	if err := m.resetRuntime(ctx, volumes, cacheBuster); err != nil {
		return nil, err
	}

	mysql := m.mysqlService(source)
	mysql, err := mysql.Start(ctx)
	if err != nil {
		return nil, err
	}
	if err := m.setupTemporalSchema(ctx, mysql, cacheBuster); err != nil {
		return nil, err
	}

	temporal := m.temporalService(source, mysql)
	if err := m.createTemporalNamespace(ctx, temporal, cacheBuster); err != nil {
		return nil, err
	}

	ambox := m.amboxService()
	ambox, err = ambox.Start(ctx)
	if err != nil {
		return nil, err
	}
	if err := m.settleAmbox(ctx, ambox, cacheBuster); err != nil {
		return nil, err
	}

	lgtm := m.lgtmService(cacheBuster)
	enduro := m.enduroRuntime(image, volumes, ambox, enduroConfig(), coverage, cacheBuster).
		WithServiceBinding("mysql", mysql).
		WithServiceBinding("temporal", temporal).
		WithServiceBinding("lgtm", lgtm).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})

	return &smokeEnvironment{
		coverage:    coverage,
		cacheBuster: cacheBuster,
		source:      source,
		volumes:     volumes,
		mysql:       mysql,
		temporal:    temporal,
		ambox:       ambox,
		enduro:      enduro,
		lgtm:        lgtm,
	}, nil
}

func (m *EnduroE2E) objectStorageEnvironment(ctx context.Context, source *dagger.Directory, image *dagger.Container, coverage bool, provider objectStorageProvider) (*smokeEnvironment, error) {
	cacheBuster := fmt.Sprintf("%s-%d", provider, time.Now().UnixNano())
	volumes := runtimeVolumes{
		batch:      dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-batch", provider)),
		watched:    dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-watched", provider)),
		completed:  dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-completed", provider)),
		transfers:  dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-transfers", provider)),
		processing: dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-processing", provider)),
		receipts:   dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-receipts", provider)),
		storage:    dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-storage", provider)),
		coverage:   dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-coverage", provider)),
	}

	if err := m.resetRuntime(ctx, volumes, cacheBuster); err != nil {
		return nil, err
	}

	mysql := m.mysqlService(source)
	mysql, err := mysql.Start(ctx)
	if err != nil {
		return nil, err
	}
	temporal := m.temporalDevService()
	if err := m.waitTemporalNamespace(ctx, temporal, cacheBuster); err != nil {
		return nil, err
	}

	redis := m.redisService()
	redis, err = redis.Start(ctx)
	if err != nil {
		return nil, err
	}

	ambox := m.amboxService()
	ambox, err = ambox.Start(ctx)
	if err != nil {
		return nil, err
	}
	if err := m.settleAmbox(ctx, ambox, cacheBuster); err != nil {
		return nil, err
	}

	var enduro *dagger.Service
	switch provider {
	case objectStorageProviderMinioLatest, objectStorageProviderArgmin:
		var storage *dagger.Service
		if provider == objectStorageProviderArgmin {
			storage = m.argminService(cacheBuster)
		} else {
			storage = m.minioService(volumes, redis)
		}
		storage, err = storage.Start(ctx)
		if err != nil {
			return nil, err
		}
		if err := m.setupObjectStorageBucket(ctx, source, storage, provider, cacheBuster); err != nil {
			return nil, err
		}
		enduro = m.enduroRuntime(image, volumes, ambox, enduroObjectStorageConfig(provider, fmt.Sprintf("http://%s:9000", provider)), coverage, cacheBuster).
			WithServiceBinding("mysql", mysql).
			WithServiceBinding("temporal", temporal).
			WithServiceBinding("redis", redis).
			WithServiceBinding(string(provider), storage).
			AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
		enduro, err = enduro.Start(ctx)
		if err != nil {
			return nil, err
		}
		return &smokeEnvironment{
			coverage:    coverage,
			cacheBuster: cacheBuster,
			source:      source,
			volumes:     volumes,
			mysql:       mysql,
			temporal:    temporal,
			ambox:       ambox,
			enduro:      enduro,
			redis:       redis,
			storage:     storage,
		}, nil
	case objectStorageProviderSeaweedFS:
		// Explicit session hostnames avoid a dependency cycle: Enduro reads S3,
		// while SeaweedFS delivers notifications to Enduro. Start both before upload.
		// Dagger appends session domains, so keep the hostname below Linux's
		// 64-byte limit even after both domain suffixes are added.
		hostSuffix := strconv.FormatInt(time.Now().UnixNano(), 36)
		enduroHost := "e-" + hostSuffix
		storageHost := "s-" + hostSuffix
		storage := dag.Container().From(seaweedFSImage).
			WithEnvVariable("AWS_ACCESS_KEY_ID", "minio").
			WithEnvVariable("AWS_SECRET_ACCESS_KEY", "minio123").
			WithNewFile("/etc/seaweedfs/notification.toml", seaweedFSNotificationConfig("http://"+enduroHost+":7480/seaweedfs/events")).
			WithExposedPort(8333).
			WithEntrypoint([]string{"weed", "mini", "-dir=/tmp/seaweedfs", "-ip.bind=0.0.0.0", "-bucket=sips", "-master.volumeSizeLimitMB=1024", "-webdav=false", "-admin.ui=false", "-s3.port.iceberg=0"}).
			AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true}).WithHostname(storageHost)
		storage, err = storage.Start(ctx)
		if err != nil {
			return nil, err
		}
		enduro = m.enduroRuntime(image, volumes, ambox, enduroObjectStorageConfig(provider, "http://"+storageHost+":8333"), coverage, cacheBuster).
			WithServiceBinding("mysql", mysql).
			WithServiceBinding("temporal", temporal).
			WithServiceBinding("redis", redis).
			WithExposedPort(7480).
			AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true}).WithHostname(enduroHost)
		enduro, err = enduro.Start(ctx)
		if err != nil {
			return nil, err
		}
		return &smokeEnvironment{
			coverage:    coverage,
			cacheBuster: cacheBuster,
			source:      source,
			volumes:     volumes,
			mysql:       mysql,
			temporal:    temporal,
			ambox:       ambox,
			enduro:      enduro,
			redis:       redis,
			storage:     storage,
		}, nil

	default:
		return nil, fmt.Errorf("unknown object storage provider %q", provider)
	}
}

// Install tools and dependencies before adding test sources or live services.
// Per-run state must not invalidate these expensive, reusable layers.
func (m *EnduroE2E) smokeTester(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From(playwrightImage).
		WithWorkdir("/e2e").
		WithEnvVariable("PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD", "1").
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends curl jq zip p7zip-full ca-certificates redis-tools && rm -rf /var/lib/apt/lists/*"}).
		WithDirectory("/e2e", source.Directory("hack/dagger/smoke"), dagger.ContainerWithDirectoryOpts{
			Include: []string{"package.json", "package-lock.json"},
		}).
		WithMountedCache("/root/.npm", dag.CacheVolume("enduro-e2e-smoke-npm-cache")).
		WithExec([]string{"npm", "ci"}).
		WithDirectory("/e2e", source.Directory("hack/dagger/smoke")).
		WithDirectory("/artifacts", dag.Directory()).
		WithDirectory("/tmp/temporal-artifacts", dag.Directory())
}

func (m *EnduroE2E) runSmokeSuite(ctx context.Context, env *smokeEnvironment) (*dagger.Directory, error) {
	temporalCLI := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		File("/usr/local/bin/temporal")

	tester := m.smokeTester(env.source).
		WithServiceBinding("enduro", env.enduro).
		WithServiceBinding("ambox", env.ambox).
		WithServiceBinding("temporal", env.temporal).
		WithServiceBinding("lgtm", env.lgtm).
		WithMountedCache("/runtime/batch", env.volumes.batch).
		WithMountedCache("/runtime/watched", env.volumes.watched).
		WithMountedCache("/runtime/completed", env.volumes.completed).
		WithMountedCache("/runtime/receipts", env.volumes.receipts).
		WithEnvVariable("E2E_CACHE_BUSTER", env.cacheBuster).
		WithEnvVariable("ENDURO_URL", "http://enduro:9000").
		WithEnvVariable("LGTM_URL", "http://lgtm:3000").
		WithEnvVariable("TEMPORAL_ADDRESS", "temporal:7233").
		WithEnvVariable("WATCHED_DIR", "/runtime/watched").
		WithEnvVariable("BATCH_DIR", "/runtime/batch").
		WithEnvVariable("RECEIPTS_DIR", "/runtime/receipts/out").
		WithEnvVariable("ARTIFACTS_DIR", "/artifacts").
		WithEnvVariable("BATCH_TRANSFER_NAME", fmt.Sprintf("avl-other-sip-2.16.578.1.39.100.11.%s-20260721", env.cacheBuster)).
		WithEnvVariable("EXPECTED_PROCESSING_WORKFLOWS", "2").
		WithEnvVariable("VERIFY_BATCH_WORKFLOW", "true").
		WithFile("/usr/local/bin/temporal", temporalCLI, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		})

	return m.finishSmokeSuite(ctx, env, tester, "tests/ambox-smoke.spec.ts")
}

func (m *EnduroE2E) runObjectStorageSmokeSuite(ctx context.Context, env *smokeEnvironment, provider objectStorageProvider) (*dagger.Directory, error) {
	temporalCLI := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		File("/usr/local/bin/temporal")

	s3put := m.s3PutBinary(env.source)
	s3Endpoint := fmt.Sprintf("http://%s:9000", provider)
	redisList := "minio-events"
	if provider == objectStorageProviderArgmin {
		redisList = "object-events"
	}
	if provider == objectStorageProviderSeaweedFS {
		s3Endpoint = "http://seaweedfs:8333"
		redisList = "object-events"
	}

	tester := m.smokeTester(env.source).
		WithServiceBinding("enduro", env.enduro).
		WithServiceBinding("temporal", env.temporal).
		WithServiceBinding("redis", env.redis).
		WithEnvVariable("E2E_CACHE_BUSTER", env.cacheBuster).
		WithEnvVariable("ENDURO_URL", "http://enduro:9000").
		WithEnvVariable("TEMPORAL_ADDRESS", "temporal:7233").
		WithEnvVariable("ARTIFACTS_DIR", "/artifacts").
		WithEnvVariable("OBJECT_STORAGE_SCENARIO", string(provider)).
		WithEnvVariable("OBJECT_STORAGE_TRANSFER_NAME", fmt.Sprintf("%s-%s.zip", provider, env.cacheBuster)).
		WithEnvVariable("S3_ENDPOINT", s3Endpoint).
		WithEnvVariable("S3_BUCKET", "sips").
		WithEnvVariable("S3_ACCESS_KEY_ID", "minio").
		WithEnvVariable("S3_SECRET_ACCESS_KEY", "minio123").
		WithEnvVariable("S3_REGION", "us-west-1").
		WithEnvVariable("REDIS_LIST", redisList).
		WithEnvVariable("OBJECT_STORAGE_UPLOAD_DELAY_MS", "45000").
		WithFile("/usr/local/bin/temporal", temporalCLI, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithFile("/usr/local/bin/s3put", s3put, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		})

	if env.storage != nil {
		tester = tester.WithServiceBinding(string(provider), env.storage)
	}

	return m.finishSmokeSuite(ctx, env, tester, "tests/object-storage-smoke.spec.ts")
}

func (m *EnduroE2E) finishSmokeSuite(ctx context.Context, env *smokeEnvironment, tester *dagger.Container, suite string) (*dagger.Directory, error) {
	tester = tester.WithExec([]string{"npx", "playwright", "test", suite, "--reporter=line"}, dagger.ContainerWithExecOpts{
		Expect: dagger.ReturnTypeAny,
	})
	code, err := tester.ExitCode(ctx)
	if err != nil {
		return nil, err
	}
	artifacts := tester.Directory("/artifacts").
		WithDirectory(".", tester.Directory("/tmp/temporal-artifacts"))
	output, err := tester.CombinedOutput(ctx)
	if err != nil {
		return nil, err
	}
	artifacts = artifacts.WithNewFile("playwright.log", output)

	// Go flushes coverage at shutdown. Retain test artifacts if shutdown or
	// coverage conversion fails, and report that failure to the caller too.
	if _, err = env.enduro.Stop(ctx); err == nil {
		var coverageArtifacts *dagger.Directory
		coverageArtifacts, err = m.convertEnduroCoverage(ctx, env, artifacts)
		if err == nil {
			artifacts = coverageArtifacts
		}
	}
	if err != nil {
		artifacts = artifacts.WithNewFile("cleanup-error.txt", err.Error()+"\n")
		if code == 0 {
			code = 1
		}
	}
	if code != 0 && !m.ReportOnly {
		return nil, fmt.Errorf("smoke suite %s failed (exit %d): %s; cleanup: %v", suite, code, output, err)
	}
	return artifacts.WithNewFile("smoke-exit-code", strconv.Itoa(code)+"\n"), nil
}

func (m *EnduroE2E) resetRuntime(ctx context.Context, volumes runtimeVolumes, cacheBuster string) error {
	reset := dag.Container().
		From("alpine:3.24.1").
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithMountedCache("/runtime/batch", volumes.batch).
		WithMountedCache("/runtime/watched", volumes.watched).
		WithMountedCache("/runtime/completed", volumes.completed).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/receipts", volumes.receipts).
		WithMountedCache("/runtime/storage", volumes.storage).
		WithMountedCache("/runtime/coverage", volumes.coverage).
		WithExec([]string{"sh", "-ceu", strings.TrimSpace(`
			rm -rf /runtime/batch/* /runtime/watched/* /runtime/completed/* /runtime/transfers/* /runtime/processing/* /runtime/receipts/* /runtime/storage/* /runtime/coverage/*
			mkdir -p /runtime/batch /runtime/watched /runtime/completed /runtime/transfers /runtime/processing /runtime/receipts /runtime/storage /runtime/coverage/raw
			chown -R 65532:65532 /runtime
		`)})

	_, err := reset.Stdout(ctx)
	return err
}

func (m *EnduroE2E) mysqlService(source *dagger.Directory) *dagger.Service {
	return dag.Container().
		From(mysqlImage).
		WithEnvVariable("MYSQL_ROOT_PASSWORD", "root123").
		WithEnvVariable("MYSQL_USER", "enduro").
		WithEnvVariable("MYSQL_PASSWORD", "enduro123").
		WithFile("/docker-entrypoint-initdb.d/docker-init.sql", source.File("hack/docker-init-mysql.sql")).
		WithExposedPort(3306).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

func (m *EnduroE2E) setupTemporalSchema(ctx context.Context, mysql *dagger.Service, cacheBuster string) error {
	setup := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		WithServiceBinding("mysql", mysql).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithExec([]string{"/bin/sh", "-ceu", temporalSchemaScript})

	_, err := setup.Stdout(ctx)
	return err
}

func (m *EnduroE2E) temporalService(source *dagger.Directory, mysql *dagger.Service) *dagger.Service {
	return withoutOTelEnv(dag.Container().
		From(temporalImage).
		WithServiceBinding("mysql", mysql).
		WithEnvVariable("DB", "mysql8").
		WithEnvVariable("DB_PORT", "3306").
		WithEnvVariable("MYSQL_USER", "enduro").
		WithEnvVariable("MYSQL_PWD", "enduro123").
		WithEnvVariable("MYSQL_SEEDS", "mysql").
		WithEnvVariable("BIND_ON_IP", "0.0.0.0").
		WithEnvVariable("DYNAMIC_CONFIG_FILE_PATH", "config/dynamicconfig/development-sql.yaml").
		WithDirectory("/etc/temporal/config/dynamicconfig", source.Directory("hack/etc/temporal/dynamicconfig")).
		WithExposedPort(7233)).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

func (m *EnduroE2E) temporalDevService() *dagger.Service {
	return withoutOTelEnv(dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		WithEntrypoint([]string{
			"temporal",
			"server",
			"start-dev",
			"--ip",
			"0.0.0.0",
			"--headless",
			"--log-level",
			"warn",
		}).
		WithExposedPort(7233)).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

func (m *EnduroE2E) createTemporalNamespace(ctx context.Context, temporal *dagger.Service, cacheBuster string) error {
	setup := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		WithServiceBinding("temporal", temporal).
		WithEnvVariable("TEMPORAL_ADDRESS", "temporal:7233").
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithExec([]string{"/bin/sh", "-ceu", temporalNamespaceScript})

	_, err := setup.Stdout(ctx)
	return err
}

func (m *EnduroE2E) waitTemporalNamespace(ctx context.Context, temporal *dagger.Service, cacheBuster string) error {
	waiter := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		WithServiceBinding("temporal", temporal).
		WithEnvVariable("TEMPORAL_ADDRESS", "temporal:7233").
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithExec([]string{"/bin/sh", "-ceu", temporalNamespaceWaitScript})

	_, err := waiter.Stdout(ctx)
	return err
}

func withoutOTelEnv(container *dagger.Container) *dagger.Container {
	for _, name := range []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
	} {
		container = container.WithoutEnvVariable(name)
	}
	return container.WithEnvVariable("OTEL_TRACES_EXPORTER", "none")
}

func (m *EnduroE2E) amboxService() *dagger.Service {
	return dag.Container().
		From(amboxImage).
		WithNewFile("/etc/sftpgo/initial-data.json", sftpgoInitialData(), dagger.ContainerWithNewFileOpts{
			Permissions: 0o644,
		}).
		WithExposedPort(64080).
		WithExposedPort(64081).
		WithExposedPort(64022).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint: true,
			NoInit:        true,
		})
}

func (m *EnduroE2E) settleAmbox(ctx context.Context, ambox *dagger.Service, cacheBuster string) error {
	waiter := dag.Container().
		From("alpine:3.24.1").
		WithServiceBinding("ambox", ambox).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithExec([]string{"sh", "-ceu", "sleep 45"})

	_, err := waiter.Stdout(ctx)
	return err
}

func (m *EnduroE2E) redisService() *dagger.Service {
	return dag.Container().
		From(redisImage).
		WithExposedPort(6379).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

func (m *EnduroE2E) lgtmService(cacheBuster string) *dagger.Service {
	return dag.Container().
		From(lgtmImage).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithEnvVariable("ENABLE_LOGS_OTELCOL", "true").
		WithEnvVariable("ENABLE_LOGS_TEMPO", "true").
		WithExposedPort(3000).
		WithExposedPort(4317).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

// Build independently of Enduro's source and per-run cache buster. Copy the
// executable out of the target cache so it is part of Dagger's immutable output.
func (m *EnduroE2E) argminBinary() *dagger.File {
	source := dag.Git("https://github.com/justincormack/argmin.git").Commit(argminRevision).Tree()
	return dag.Container().
		From(argminBuildImage).
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/usr/local/cargo/registry", dag.CacheVolume("enduro-argmin-cargo-registry")).
		WithMountedCache("/usr/local/cargo/git", dag.CacheVolume("enduro-argmin-cargo-git")).
		WithMountedCache("/src/target", dag.CacheVolume("enduro-argmin-cargo-target"), dagger.ContainerWithMountedCacheOpts{
			Sharing: dagger.CacheSharingModeLocked,
		}).
		WithExec([]string{"cargo", "build", "--locked", "--release", "-p", "argmin-s3"}).
		WithExec([]string{"install", "-D", "target/release/argmin-s3", "/out/argmin-s3"}).
		File("/out/argmin-s3")
}

func (m *EnduroE2E) argminService(cacheBuster string) *dagger.Service {
	return dag.Container().
		From("debian:bookworm-slim").
		WithFile("/usr/local/bin/argmin-s3", m.argminBinary()).
		WithExec([]string{"sh", "-ceu", "mkdir /data && chown 1000:1000 /data"}).
		WithUser("1000:1000").
		WithEnvVariable("ARGMIN_ACCOUNT_ID", "111122223333").
		WithEnvVariable("ARGMIN_ACCESS_KEY_ID", "minio").
		WithEnvVariable("ARGMIN_SECRET_ACCESS_KEY", "minio123").
		// Fixed test-only key; each run has fresh ephemeral storage.
		WithEnvVariable("ARGMIN_SSE_S3_WRAPPING_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=").
		WithEnvVariable("ARGMIN_REGION", "us-west-1").
		WithEnvVariable("ARGMIN_DATA_DIR", "/data").
		WithEnvVariable("ARGMIN_LISTEN_ADDR", "0.0.0.0:9000").
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithExposedPort(9000).
		WithDefaultArgs([]string{"argmin-s3"}).
		AsService()
}

func (m *EnduroE2E) minioService(volumes runtimeVolumes, redis *dagger.Service) *dagger.Service {
	return dag.Container().
		From(minioLatestImage).
		WithServiceBinding("redis", redis).
		WithMountedCache("/storage", volumes.storage).
		WithEnvVariable("MINIO_ROOT_USER", "minio").
		WithEnvVariable("MINIO_ROOT_PASSWORD", "minio123").
		WithEnvVariable("MINIO_NOTIFY_REDIS_ENABLE_PRIMARY", "on").
		WithEnvVariable("MINIO_NOTIFY_REDIS_ADDRESS_PRIMARY", "redis:6379").
		WithEnvVariable("MINIO_NOTIFY_REDIS_KEY_PRIMARY", "minio-events").
		WithEnvVariable("MINIO_NOTIFY_REDIS_FORMAT_PRIMARY", "access").
		WithEnvVariable("MINIO_NOTIFY_REDIS_QUEUE_DIR_PRIMARY", "/tmp/events").
		WithEnvVariable("MINIO_NOTIFY_REDIS_QUEUE_LIMIT_PRIMARY", "10000").
		WithEnvVariable("MINIO_BROWSER_LOGIN_ANIMATION", "off").
		WithExposedPort(9000).
		WithExposedPort(9001).
		WithDefaultArgs([]string{"server", "--console-address", ":9001", "/storage"}).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

func (m *EnduroE2E) setupObjectStorageBucket(
	ctx context.Context,
	source *dagger.Directory,
	storage *dagger.Service,
	provider objectStorageProvider,
	cacheBuster string,
) error {
	notificationARN := ""
	if isMinIOProvider(provider) {
		notificationARN = "arn:minio:sqs::PRIMARY:redis"
	}
	setup := dag.Container().
		From(goImage).
		WithServiceBinding(string(provider), storage).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithFile("/usr/local/bin/s3setup", m.s3SetupBinary(source), dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithExec([]string{
			"s3setup",
			"-endpoint", fmt.Sprintf("http://%s:9000", provider),
			"-region", "us-west-1",
			"-bucket", "sips",
			"-access-key", "minio",
			"-secret-key", "minio123",
			"-notification-arn", notificationARN,
		})

	_, err := setup.Stdout(ctx)
	return err
}

// runtimeImage selects a release artifact or builds the coverage binary into
// the same Dockerfile used by GoReleaser. Runtime data never enters the build.
func (m *EnduroE2E) runtimeImage(ctx context.Context, source *dagger.Directory, archive *dagger.File, ref string, assets *dagger.Directory) (*dagger.Container, bool, error) {
	if archive != nil && ref != "" {
		return nil, false, fmt.Errorf("image-archive and image-ref are mutually exclusive")
	}
	if (archive != nil || ref != "") && assets != nil {
		return nil, false, fmt.Errorf("frontend-assets only applies to source coverage builds")
	}
	if archive != nil {
		return dag.Container().Import(archive), false, nil
	}
	if ref != "" {
		if !regexp.MustCompile(`^.+@sha256:[a-f0-9]{64}$`).MatchString(ref) {
			return nil, false, fmt.Errorf("image-ref must be an immutable sha256 registry reference")
		}
		return dag.Container().From(ref), false, nil
	}
	platform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, false, err
	}
	binary := m.enduroBase(source, assets).File("/src/build/enduro")
	build := dag.Directory().
		WithFile("Dockerfile", source.File("Dockerfile")).
		WithFile("LICENSE", source.File("LICENSE")).
		WithFile(string(platform)+"/enduro", binary)
	return build.DockerBuild(dagger.DirectoryDockerBuildOpts{
		Platform:  platform,
		BuildArgs: []dagger.BuildArg{{Name: "TARGETPLATFORM", Value: string(platform)}},
	}), true, nil
}

func (m *EnduroE2E) enduroRuntime(image *dagger.Container, volumes runtimeVolumes, ambox *dagger.Service, config string, coverage bool, cacheBuster string) *dagger.Container {
	knownHosts := dag.Container().
		From("alpine:3.24.1").
		WithExec([]string{"apk", "add", "--no-cache", "openssh-client"}).
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithServiceBinding("ambox", ambox).
		WithExec([]string{"sh", "-ceu", strings.TrimSpace(`
            for i in $(seq 1 90); do
                if ssh-keyscan -p 64022 ambox > /known_hosts 2>/dev/null && test -s /known_hosts; then
                    exit 0
                fi
                sleep 2
            done
            echo "timed out waiting for ambox SFTP host key" >&2
            exit 1
        `)}).
		File("/known_hosts")
	runtime := image.
		WithServiceBinding("ambox", ambox).
		WithMountedCache("/runtime/batch", volumes.batch).
		WithMountedCache("/runtime/watched", volumes.watched).
		WithMountedCache("/runtime/completed", volumes.completed).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/receipts", volumes.receipts).
		WithMountedCache("/runtime/storage", volumes.storage).
		WithNewFile("/etc/enduro/enduro.toml", config, dagger.ContainerWithNewFileOpts{Permissions: 0o644}).
		WithNewFile("/etc/enduro/ssh/sftp_key", e2eSFTPPrivateKey, dagger.ContainerWithNewFileOpts{Permissions: 0o600, Owner: "65532:65532"}).
		WithFile("/etc/enduro/ssh/known_hosts", knownHosts, dagger.ContainerWithFileOpts{Permissions: 0o644}).
		WithExposedPort(9000).
		WithExposedPort(9001)
	if coverage {
		runtime = runtime.
			WithMountedCache("/runtime/coverage", volumes.coverage).
			WithEnvVariable("GOCOVERDIR", "/runtime/coverage/raw")
	}
	return runtime
}

func (m *EnduroE2E) enduroBase(source, frontendAssets *dagger.Directory) *dagger.Container {
	if frontendAssets == nil {
		frontendAssets = m.frontendAssets(source)
	}

	return dag.Container().
		From(goImage).
		WithDirectory("/src", source, dagger.ContainerWithDirectoryOpts{
			Exclude: []string{
				".git",
				"build",
				"dist",
				"frontend/.nuxt",
				"frontend/.output",
				"**/node_modules",
				"frontend/coverage",
				"hack/dagger/runtime",
				"hack/minio-data",
				"hack/seaweedfs-data",
			},
		}).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "0").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("enduro-e2e-go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("enduro-e2e-go-build")).
		WithDirectory("/src/frontend/.output/public", frontendAssets).
		WithExec([]string{"go", "build", "-cover", "-covermode=atomic", "-coverpkg=./...", "-trimpath", "-o", "build/enduro", "."})
}

func (m *EnduroE2E) frontendAssets(source *dagger.Directory) *dagger.Directory {
	return dag.Container().
		From(nodeImage).
		WithWorkdir("/src/frontend").
		WithDirectory("/src/frontend", source.Directory("frontend"), dagger.ContainerWithDirectoryOpts{
			Include: []string{"package.json", "package-lock.json"},
		}).
		WithMountedCache("/root/.npm", dag.CacheVolume("enduro-e2e-npm-cache")).
		WithExec([]string{"npm", "ci", "--ignore-scripts"}).
		WithDirectory("/src/frontend", source.Directory("frontend"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{
				".nuxt",
				".output",
				"coverage",
				"node_modules",
			},
		}).
		// Replay install hooks, including nuxt prepare, once sources are present.
		WithExec([]string{"npm", "rebuild"}).
		WithExec([]string{"npm", "run", "build"}).
		Directory("/src/frontend/.output/public")
}

func (m *EnduroE2E) s3PutBinary(source *dagger.Directory) *dagger.File {
	return dag.Container().
		From(goImage).
		WithDirectory("/src", source, dagger.ContainerWithDirectoryOpts{
			Exclude: []string{
				".git",
				"build",
				"dist",
				"frontend/.nuxt",
				"frontend/.output",
				"**/node_modules",
				"frontend/coverage",
				"hack/dagger/runtime",
				"hack/minio-data",
				"hack/seaweedfs-data",
			},
		}).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("enduro-e2e-go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("enduro-e2e-go-build")).
		WithExec([]string{"go", "build", "-trimpath", "-o", "/usr/local/bin/s3put", "./hack/s3put"}).
		File("/usr/local/bin/s3put")
}

func (m *EnduroE2E) s3SetupBinary(source *dagger.Directory) *dagger.File {
	return dag.Container().
		From(goImage).
		WithDirectory("/src", source, dagger.ContainerWithDirectoryOpts{
			Exclude: []string{
				".git",
				"build",
				"dist",
				"frontend/.nuxt",
				"frontend/.output",
				"**/node_modules",
				"frontend/coverage",
				"hack/dagger/runtime",
				"hack/minio-data",
				"hack/seaweedfs-data",
			},
		}).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("enduro-e2e-go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("enduro-e2e-go-build")).
		WithExec([]string{"go", "build", "-trimpath", "-o", "/usr/local/bin/s3setup", "./hack/s3setup"}).
		File("/usr/local/bin/s3setup")
}

func (m *EnduroE2E) convertEnduroCoverage(
	ctx context.Context,
	env *smokeEnvironment,
	artifacts *dagger.Directory,
) (*dagger.Directory, error) {
	if !env.coverage {
		return artifacts, nil
	}
	converter := dag.Container().
		From(goImage).
		WithMountedCache("/runtime/coverage", env.volumes.coverage).
		WithDirectory("/artifacts", artifacts).
		WithExec([]string{"sh", "-ceu", strings.TrimSpace(`
			test -n "$(find /runtime/coverage/raw -type f -name 'cov*' -print -quit)"
			go tool covdata textfmt -i=/runtime/coverage/raw -o=/artifacts/enduro-smoke.coverprofile
			go tool covdata percent -i=/runtime/coverage/raw > /artifacts/enduro-smoke-coverage.txt
		`)})

	if _, err := converter.Stdout(ctx); err != nil {
		return nil, err
	}

	return converter.Directory("/artifacts"), nil
}

func enduroConfig() string {
	return `
verbosity = 2
debug = true
debugListen = "0.0.0.0:9001"

[telemetry.traces]
enabled = true
address = "lgtm:4317"
ratio = 1.0

[temporal]
namespace = "default"
address = "temporal:7233"
taskQueue = "global"

[api]
listen = "0.0.0.0:9000"
debug = false

[database]
dsn = "enduro:enduro123@tcp(mysql:3306)/enduro"

[extractActivity]
dirMode = "0o755"
fileMode = "0o644"

[[watcher.filesystem]]
name = "e2e-fs"
path = "/runtime/watched"
inotify = true
pipeline = "ambox"
completedDir = "/runtime/completed"
ignore = '(^\.gitkeep)|(^.*\.part$)'
stripTopLevelDir = true
rejectDuplicates = false
excludeHiddenFiles = false
transferType = "standard"

[[pipeline]]
name = "ambox"
baseURL = "http://ambox:64080"
user = "test"
key = "test"
transferDir = "/runtime/transfers"
processingDir = "/runtime/processing"
processingConfig = "automated"
storageServiceURL = "http://test:test@ambox:64081"
capacity = 1
retryDeadline = "10m"
statusRequestTimeout = "30s"
transferDeadline = "2h"
unbag = false

[pipeline.transferPublisher]
type = "sftp"
host = "ambox"
port = 64022
user = "archivematica"
remoteDir = "/"
submittedPathPrefix = "archivematica/transfers"
knownHostsFile = "/etc/enduro/ssh/known_hosts"

[pipeline.transferPublisher.privateKey]
path = "/etc/enduro/ssh/sftp_key"
passphrase = ""

[pipeline.recovery]
reconcileExistingAIP = true

[validation]
checksumsCheckEnabled = false

[[hooks."hari"]]
baseURL = ""
mock = true
disabled = true

[[hooks."prod"]]
receiptPath = "/runtime/receipts/out"
disabled = false

[metadata]
processNameMetadata = false

[worker]
heartbeatThrottleInterval = "1m"
maxConcurrentWorkflowsExecutionsSize = 10
maxConcurrentSessionExecutionSize = 4

[workflow]
activityHeartbeatTimeout = "30s"
`
}

func enduroObjectStorageConfig(provider objectStorageProvider, endpoint string) string {
	var webhookConfig string
	eventFormat := "minio"
	redisList := "minio-events"
	if provider == objectStorageProviderArgmin {
		eventFormat = "enduro"
		redisList = "object-events"
	}
	if provider == objectStorageProviderSeaweedFS {
		eventFormat = "enduro"
		redisList = "object-events"
		webhookConfig = `
[objectEventWebhook]
enabled = true
listen = "0.0.0.0:7480"
redisAddress = "redis://redis:6379"
redisList = "object-events"
bucketsPath = "/buckets"
`
	}

	return fmt.Sprintf(`
verbosity = 2
debug = true
debugListen = "0.0.0.0:9001"

[telemetry.traces]
enabled = false
address = ""
ratio = 1.0

[temporal]
namespace = "default"
address = "temporal:7233"
taskQueue = "global"

[api]
listen = "0.0.0.0:9000"
debug = false

[database]
dsn = "enduro:enduro123@tcp(mysql:3306)/enduro"

[extractActivity]
dirMode = "0o755"
fileMode = "0o644"
%s
[[watcher.s3]]
name = "e2e-%s"
eventSource = "redis"
eventFormat = %q
redisAddress = "redis://redis:6379"
redisList = %q
endpoint = %q
pathStyle = true
key = "minio"
secret = "minio123"
region = "us-west-1"
bucket = "sips"
pipeline = "ambox"
retentionPeriod = "10s"
stripTopLevelDir = true
rejectDuplicates = false
excludeHiddenFiles = false
transferType = "standard"

[[pipeline]]
name = "ambox"
baseURL = "http://ambox:64080"
user = "test"
key = "test"
transferDir = "/runtime/transfers"
processingDir = "/runtime/processing"
processingConfig = "automated"
storageServiceURL = "http://test:test@ambox:64081"
capacity = 1
retryDeadline = "10m"
statusRequestTimeout = "30s"
transferDeadline = "2h"
unbag = false

[pipeline.transferPublisher]
type = "sftp"
host = "ambox"
port = 64022
user = "archivematica"
remoteDir = "/"
submittedPathPrefix = "archivematica/transfers"
knownHostsFile = "/etc/enduro/ssh/known_hosts"

[pipeline.transferPublisher.privateKey]
path = "/etc/enduro/ssh/sftp_key"
passphrase = ""

[pipeline.recovery]
reconcileExistingAIP = true

[validation]
checksumsCheckEnabled = false

[[hooks."hari"]]
baseURL = ""
mock = true
disabled = true

[[hooks."prod"]]
receiptPath = ""
disabled = true

[metadata]
processNameMetadata = false

[worker]
heartbeatThrottleInterval = "1m"
maxConcurrentWorkflowsExecutionsSize = 2
maxConcurrentSessionExecutionSize = 1

[workflow]
activityHeartbeatTimeout = "5m"
initProcessingTimeout = "1m"
`, webhookConfig, provider, eventFormat, redisList, endpoint)
}

func isMinIOProvider(provider objectStorageProvider) bool {
	return provider == objectStorageProviderMinioLatest
}

func seaweedFSNotificationConfig(endpoint string) string {
	return fmt.Sprintf(`
[notification.webhook]
enabled = true
endpoint = %q
timeout_seconds = 10
max_retries = 3
backoff_seconds = 3
max_backoff_seconds = 30
workers = 2
buffer_size = 1000
event_types = ["create"]
path_prefixes = ["/buckets/sips/"]
`, endpoint)
}

const temporalSchemaScript = `
for i in $(seq 1 90); do
	if temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal setup-schema -v 0.0; then
		break
	fi
	if [ "$i" = "90" ]; then
		echo "timed out waiting for mysql before Temporal schema setup" >&2
		exit 1
	fi
	sleep 2
done

temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal update-schema --schema-name mysql/v8/temporal
temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal_visibility setup-schema -v 0.0
temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal_visibility update-schema --schema-name mysql/v8/visibility
`

const temporalNamespaceScript = `
for i in $(seq 1 90); do
	if temporal operator namespace create --namespace default --retention 72h; then
		exit 0
	fi
	sleep 2
done

echo "timed out creating Temporal namespace" >&2
exit 1
`

const temporalNamespaceWaitScript = `
for i in $(seq 1 90); do
	if temporal operator namespace describe --namespace default; then
		exit 0
	fi
	sleep 2
done

echo "timed out waiting for Temporal namespace" >&2
exit 1
`

func sftpgoInitialData() string {
	return fmt.Sprintf(`{
  "users": [
    {
      "id": 1,
      "status": 1,
      "username": "archivematica",
      "password": "12345",
      "public_keys": [
        %q
      ],
      "has_password": true,
      "home_dir": "/home/archivematica/transfers",
      "permissions": { "/": ["*"] }
    }
  ]
}
`, e2eSFTPPublicKey)
}

const e2eSFTPPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJ/IK3nGsm/nL26tNTW4xjgwFzNFABH+AfdsUY1HY6YO enduro-e2e"

const e2eSFTPPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCfyCt5xrJv5y9urTU1uMY4MBczRQAR/gH3bFGNR2OmDgAAAJDZ6m3o2ept
6AAAAAtzc2gtZWQyNTUxOQAAACCfyCt5xrJv5y9urTU1uMY4MBczRQAR/gH3bFGNR2OmDg
AAAEDZHrw95M/Mb43weiEcSIHWBirLfrtJN+eXTeXpB4mZcp/IK3nGsm/nL26tNTW4xjgw
FzNFABH+AfdsUY1HY6YOAAAACmVuZHVyby1lMmUBAgM=
-----END OPENSSH PRIVATE KEY-----
`
