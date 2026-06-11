// Enduro integration-test harnesses.
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dagger/enduro-e-2-e/internal/dagger"
)

const (
	mysqlImage        = "mysql:8.4.8-oraclelinux9"
	temporalImage     = "temporalio/server:1.30.3"
	temporalToolImage = "temporalio/admin-tools:1.30.3"
	goImage           = "golang:1.26.4-bookworm"
	nodeImage         = "node:24-bookworm"
	redisImage        = "redis:8.2.6-alpine3.22"
	minioLegacyImage  = "minio/minio:RELEASE.2020-04-28T23-56-56Z"
	minioLatestImage  = "minio/minio:RELEASE.2025-09-07T16-13-09Z"
	seaweedFSImage    = "chrislusf/seaweedfs:4.31"
	amboxImage        = "ghcr.io/sevein/ambox:latest"
	playwrightImage   = "mcr.microsoft.com/playwright:v1.60.0-noble"
)

type EnduroE2E struct{}

type objectStorageProvider string

const (
	objectStorageProviderMinioLegacy objectStorageProvider = "minio-legacy"
	objectStorageProviderMinioLatest objectStorageProvider = "minio-latest"
	objectStorageProviderSeaweedFS   objectStorageProvider = "seaweedfs"
)

type runtimeVolumes struct {
	batch      *dagger.CacheVolume
	watched    *dagger.CacheVolume
	completed  *dagger.CacheVolume
	transfers  *dagger.CacheVolume
	processing *dagger.CacheVolume
	storage    *dagger.CacheVolume
	coverage   *dagger.CacheVolume
}

type smokeEnvironment struct {
	cacheBuster string
	source      *dagger.Directory
	volumes     runtimeVolumes
	mysql       *dagger.Service
	temporal    *dagger.Service
	ambox       *dagger.Service
	enduro      *dagger.Service
	redis       *dagger.Service
	storage     *dagger.Service
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
//	dagger -m hack/dagger call smoke-tests --source . export --path hack/dagger/runtime/artifacts
func (m *EnduroE2E) SmokeTests(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	env, err := m.smokeEnvironment(ctx, source)
	if err != nil {
		return nil, err
	}

	return m.runSmokeSuite(ctx, env)
}

// Run S3 watcher smoke tests against MinIO and SeaweedFS.
//
// The MinIO scenarios exercise native MinIO Redis notifications with both the
// legacy fixture-backed image and the newer env-configured image. The SeaweedFS
// scenario exercises SeaweedFS filer webhooks, Enduro's object event webhook,
// normalized Redis events, and the S3 watcher path. All scenarios publish the
// transfer to ambox and verify the resulting AIP.
//
// Call from the repository root with:
//
//	dagger -m hack/dagger call object-storage-smoke-tests --source . export --path hack/dagger/runtime/object-storage-artifacts
func (m *EnduroE2E) ObjectStorageSmokeTests(ctx context.Context, source *dagger.Directory) (*dagger.Directory, error) {
	minioLegacyArtifacts, err := m.objectStorageSmokeTest(ctx, source, objectStorageProviderMinioLegacy)
	if err != nil {
		return nil, err
	}

	minioLatestArtifacts, err := m.objectStorageSmokeTest(ctx, source, objectStorageProviderMinioLatest)
	if err != nil {
		return nil, err
	}

	seaweedFSArtifacts, err := m.objectStorageSmokeTest(ctx, source, objectStorageProviderSeaweedFS)
	if err != nil {
		return nil, err
	}

	return dag.Directory().
		WithDirectory(string(objectStorageProviderMinioLegacy), minioLegacyArtifacts).
		WithDirectory(string(objectStorageProviderMinioLatest), minioLatestArtifacts).
		WithDirectory("seaweedfs", seaweedFSArtifacts), nil
}

// Run one S3 watcher smoke test provider.
//
// Provider must be "minio-legacy", "minio-latest", or "seaweedfs".
func (m *EnduroE2E) ObjectStorageSmokeTest(ctx context.Context, source *dagger.Directory, provider string) (*dagger.Directory, error) {
	switch objectStorageProvider(provider) {
	case objectStorageProviderMinioLegacy:
		return m.objectStorageSmokeTest(ctx, source, objectStorageProviderMinioLegacy)
	case objectStorageProviderMinioLatest:
		return m.objectStorageSmokeTest(ctx, source, objectStorageProviderMinioLatest)
	case objectStorageProviderSeaweedFS:
		return m.objectStorageSmokeTest(ctx, source, objectStorageProviderSeaweedFS)
	default:
		return nil, fmt.Errorf("unknown object storage provider %q", provider)
	}
}

func (m *EnduroE2E) objectStorageSmokeTest(ctx context.Context, source *dagger.Directory, provider objectStorageProvider) (*dagger.Directory, error) {
	env, err := m.objectStorageEnvironment(ctx, source, provider)
	if err != nil {
		return nil, err
	}

	return m.runObjectStorageSmokeSuite(ctx, env, provider)
}

func (m *EnduroE2E) smokeEnvironment(ctx context.Context, source *dagger.Directory) (*smokeEnvironment, error) {
	cacheBuster := fmt.Sprintf("%d", time.Now().UnixNano())
	volumes := runtimeVolumes{
		batch:      dag.CacheVolume("enduro-e2e-ambox-batch"),
		watched:    dag.CacheVolume("enduro-e2e-ambox-watched"),
		completed:  dag.CacheVolume("enduro-e2e-ambox-completed"),
		transfers:  dag.CacheVolume("enduro-e2e-ambox-transfers"),
		processing: dag.CacheVolume("enduro-e2e-ambox-processing"),
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

	enduro := m.enduroService(source, volumes, mysql, temporal, ambox)

	return &smokeEnvironment{
		cacheBuster: cacheBuster,
		source:      source,
		volumes:     volumes,
		mysql:       mysql,
		temporal:    temporal,
		ambox:       ambox,
		enduro:      enduro,
	}, nil
}

func (m *EnduroE2E) objectStorageEnvironment(ctx context.Context, source *dagger.Directory, provider objectStorageProvider) (*smokeEnvironment, error) {
	cacheBuster := fmt.Sprintf("%s-%d", provider, time.Now().UnixNano())
	volumes := runtimeVolumes{
		batch:      dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-batch", provider)),
		watched:    dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-watched", provider)),
		completed:  dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-completed", provider)),
		transfers:  dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-transfers", provider)),
		processing: dag.CacheVolume(fmt.Sprintf("enduro-e2e-%s-processing", provider)),
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
	case objectStorageProviderMinioLegacy, objectStorageProviderMinioLatest:
		minio := m.minioService(source, volumes, redis, provider)
		minio, err = minio.Start(ctx)
		if err != nil {
			return nil, err
		}
		if provider == objectStorageProviderMinioLatest {
			if err := m.setupMinIOBucketNotification(ctx, source, minio, provider, cacheBuster); err != nil {
				return nil, err
			}
		}
		enduro = m.enduroObjectStorageService(source, volumes, mysql, temporal, ambox, redis, minio, provider)
		enduro, err = enduro.Start(ctx)
		if err != nil {
			return nil, err
		}
		return &smokeEnvironment{
			cacheBuster: cacheBuster,
			source:      source,
			volumes:     volumes,
			mysql:       mysql,
			temporal:    temporal,
			ambox:       ambox,
			enduro:      enduro,
			redis:       redis,
			storage:     minio,
		}, nil
	case objectStorageProviderSeaweedFS:
		enduro = m.enduroSeaweedFSService(source, volumes, mysql, temporal, ambox, redis)
		enduro, err = enduro.Start(ctx)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown object storage provider %q", provider)
	}

	return &smokeEnvironment{
		cacheBuster: cacheBuster,
		source:      source,
		volumes:     volumes,
		mysql:       mysql,
		temporal:    temporal,
		ambox:       ambox,
		enduro:      enduro,
		redis:       redis,
	}, nil
}

func (m *EnduroE2E) runSmokeSuite(ctx context.Context, env *smokeEnvironment) (*dagger.Directory, error) {
	temporalCLI := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		File("/usr/local/bin/temporal")

	tester := dag.Container().
		From(playwrightImage).
		WithWorkdir("/e2e").
		WithServiceBinding("enduro", env.enduro).
		WithServiceBinding("ambox", env.ambox).
		WithServiceBinding("temporal", env.temporal).
		WithMountedCache("/runtime/batch", env.volumes.batch).
		WithMountedCache("/runtime/watched", env.volumes.watched).
		WithMountedCache("/runtime/completed", env.volumes.completed).
		WithDirectory("/e2e", env.source.Directory("hack/dagger/smoke")).
		WithEnvVariable("E2E_CACHE_BUSTER", env.cacheBuster).
		WithEnvVariable("ENDURO_URL", "http://enduro:9000").
		WithEnvVariable("TEMPORAL_ADDRESS", "temporal:7233").
		WithEnvVariable("WATCHED_DIR", "/runtime/watched").
		WithEnvVariable("BATCH_DIR", "/runtime/batch").
		WithEnvVariable("ARTIFACTS_DIR", "/artifacts").
		WithEnvVariable("BATCH_TRANSFER_NAME", fmt.Sprintf("batch-%s", env.cacheBuster)).
		WithEnvVariable("EXPECTED_PROCESSING_WORKFLOWS", "2").
		WithEnvVariable("VERIFY_BATCH_WORKFLOW", "true").
		WithEnvVariable("PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD", "1").
		WithFile("/usr/local/bin/temporal", temporalCLI, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends curl jq zip p7zip-full ca-certificates && rm -rf /var/lib/apt/lists/*"}).
		WithExec([]string{"npm", "install"})

	tester = tester.WithExec([]string{"npx", "playwright", "test", "tests/ambox-smoke.spec.ts", "--reporter=line"})

	if _, err := tester.Stdout(ctx); err != nil {
		return nil, err
	}

	// Go writes binary coverage data when the process exits, so stop Enduro
	// gracefully before converting the raw GOCOVERDIR files.
	if _, err := env.enduro.Stop(ctx); err != nil {
		return nil, err
	}

	artifacts := tester.Directory("/artifacts").
		WithDirectory(".", tester.Directory("/tmp/temporal-artifacts"))

	return m.convertEnduroCoverage(ctx, env, artifacts)
}

func (m *EnduroE2E) runObjectStorageSmokeSuite(ctx context.Context, env *smokeEnvironment, provider objectStorageProvider) (*dagger.Directory, error) {
	temporalCLI := dag.Container().
		From(temporalToolImage).
		WithoutEntrypoint().
		File("/usr/local/bin/temporal")

	s3put := m.s3PutBinary(env.source)
	s3Endpoint := fmt.Sprintf("http://%s:9000", provider)
	redisList := "minio-events"
	if provider == objectStorageProviderSeaweedFS {
		s3Endpoint = "http://enduro:8333"
		redisList = "object-events"
	}

	tester := dag.Container().
		From(playwrightImage).
		WithWorkdir("/e2e").
		WithServiceBinding("enduro", env.enduro).
		WithServiceBinding("temporal", env.temporal).
		WithServiceBinding("redis", env.redis).
		WithDirectory("/e2e", env.source.Directory("hack/dagger/smoke")).
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
		WithEnvVariable("PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD", "1").
		WithFile("/usr/local/bin/temporal", temporalCLI, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithFile("/usr/local/bin/s3put", s3put, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends curl jq zip p7zip-full ca-certificates redis-tools && rm -rf /var/lib/apt/lists/*"}).
		WithExec([]string{"npm", "install"})

	if isMinIOProvider(provider) && env.storage != nil {
		tester = tester.WithServiceBinding(string(provider), env.storage)
	}

	tester = tester.WithExec([]string{"npx", "playwright", "test", "tests/object-storage-smoke.spec.ts", "--reporter=line"})

	if _, err := tester.Stdout(ctx); err != nil {
		return nil, err
	}

	if _, err := env.enduro.Stop(ctx); err != nil {
		return nil, err
	}

	artifacts := tester.Directory("/artifacts").
		WithDirectory(".", tester.Directory("/tmp/temporal-artifacts"))

	return m.convertEnduroCoverage(ctx, env, artifacts)
}

func (m *EnduroE2E) resetRuntime(ctx context.Context, volumes runtimeVolumes, cacheBuster string) error {
	reset := dag.Container().
		From("alpine:3.22").
		WithEnvVariable("E2E_CACHE_BUSTER", cacheBuster).
		WithMountedCache("/runtime/batch", volumes.batch).
		WithMountedCache("/runtime/watched", volumes.watched).
		WithMountedCache("/runtime/completed", volumes.completed).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/storage", volumes.storage).
		WithMountedCache("/runtime/coverage", volumes.coverage).
		WithExec([]string{"sh", "-ceu", strings.TrimSpace(`
			rm -rf /runtime/batch/* /runtime/watched/* /runtime/completed/* /runtime/transfers/* /runtime/processing/* /runtime/storage/* /runtime/coverage/*
			mkdir -p /runtime/batch /runtime/watched /runtime/completed /runtime/transfers /runtime/processing /runtime/storage /runtime/coverage/raw
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
		From("alpine:3.22").
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

func (m *EnduroE2E) minioService(source *dagger.Directory, volumes runtimeVolumes, redis *dagger.Service, provider objectStorageProvider) *dagger.Service {
	var minio *dagger.Container
	switch provider {
	case objectStorageProviderMinioLegacy:
		minio = dag.Container().
			From(minioLegacyImage).
			WithServiceBinding("redis", redis).
			WithDirectory("/data", source.Directory("hack/minio-data")).
			WithExposedPort(9000).
			WithDefaultArgs([]string{"server", "/data"})
	case objectStorageProviderMinioLatest:
		minio = dag.Container().
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
			WithDefaultArgs([]string{"server", "--console-address", ":9001", "/storage"})
	default:
		panic(fmt.Sprintf("unknown MinIO provider %q", provider))
	}

	return minio.AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true})
}

func (m *EnduroE2E) setupMinIOBucketNotification(
	ctx context.Context,
	source *dagger.Directory,
	minio *dagger.Service,
	provider objectStorageProvider,
	cacheBuster string,
) error {
	setup := dag.Container().
		From(goImage).
		WithServiceBinding(string(provider), minio).
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
			"-notification-arn", "arn:minio:sqs::PRIMARY:redis",
		})

	_, err := setup.Stdout(ctx)
	return err
}

func (m *EnduroE2E) enduroService(
	source *dagger.Directory,
	volumes runtimeVolumes,
	mysql *dagger.Service,
	temporal *dagger.Service,
	ambox *dagger.Service,
) *dagger.Service {
	enduroBin := m.enduroBase(source, volumes).File("/src/build/enduro")

	return dag.Container().
		From(goImage).
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends openssh-client ca-certificates && rm -rf /var/lib/apt/lists/*"}).
		WithServiceBinding("mysql", mysql).
		WithServiceBinding("temporal", temporal).
		WithServiceBinding("ambox", ambox).
		WithMountedCache("/runtime/batch", volumes.batch).
		WithMountedCache("/runtime/watched", volumes.watched).
		WithMountedCache("/runtime/completed", volumes.completed).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/storage", volumes.storage).
		WithMountedCache("/runtime/coverage", volumes.coverage).
		WithEnvVariable("GOCOVERDIR", "/runtime/coverage/raw").
		WithFile("/usr/local/bin/enduro", enduroBin, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithNewFile("/etc/enduro/enduro.ambox.toml", enduroConfig(), dagger.ContainerWithNewFileOpts{
			Permissions: 0o644,
		}).
		WithNewFile("/etc/enduro/ssh/sftp_key", e2eSFTPPrivateKey, dagger.ContainerWithNewFileOpts{
			Permissions: 0o600,
		}).
		WithNewFile("/usr/local/bin/enduro-e2e-entrypoint", enduroEntrypointScript, dagger.ContainerWithNewFileOpts{
			Permissions: 0o755,
		}).
		WithExposedPort(9000).
		WithExposedPort(9001).
		WithEntrypoint([]string{"/usr/local/bin/enduro-e2e-entrypoint"}).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint: true,
		})
}

func (m *EnduroE2E) enduroObjectStorageService(
	source *dagger.Directory,
	volumes runtimeVolumes,
	mysql *dagger.Service,
	temporal *dagger.Service,
	ambox *dagger.Service,
	redis *dagger.Service,
	storage *dagger.Service,
	provider objectStorageProvider,
) *dagger.Service {
	enduroBin := m.enduroBase(source, volumes).File("/src/build/enduro")

	return dag.Container().
		From(goImage).
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends openssh-client curl ca-certificates && rm -rf /var/lib/apt/lists/*"}).
		WithServiceBinding("mysql", mysql).
		WithServiceBinding("temporal", temporal).
		WithServiceBinding("ambox", ambox).
		WithServiceBinding("redis", redis).
		WithServiceBinding(string(provider), storage).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/coverage", volumes.coverage).
		WithEnvVariable("GOCOVERDIR", "/runtime/coverage/raw").
		WithFile("/usr/local/bin/enduro", enduroBin, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithNewFile("/etc/enduro/enduro.ambox.toml", enduroObjectStorageConfig(provider, fmt.Sprintf("http://%s:9000", provider)), dagger.ContainerWithNewFileOpts{
			Permissions: 0o644,
		}).
		WithNewFile("/etc/enduro/ssh/sftp_key", e2eSFTPPrivateKey, dagger.ContainerWithNewFileOpts{
			Permissions: 0o600,
		}).
		WithNewFile("/usr/local/bin/enduro-e2e-entrypoint", enduroEntrypointScript, dagger.ContainerWithNewFileOpts{
			Permissions: 0o755,
		}).
		WithExposedPort(9000).
		WithExposedPort(9001).
		WithEntrypoint([]string{"/usr/local/bin/enduro-e2e-entrypoint"}).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint: true,
		})
}

func (m *EnduroE2E) enduroSeaweedFSService(
	source *dagger.Directory,
	volumes runtimeVolumes,
	mysql *dagger.Service,
	temporal *dagger.Service,
	ambox *dagger.Service,
	redis *dagger.Service,
) *dagger.Service {
	enduroBin := m.enduroBase(source, volumes).File("/src/build/enduro")
	weedBin := dag.Container().From(seaweedFSImage).File("/usr/bin/weed")

	return dag.Container().
		From(goImage).
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends openssh-client ca-certificates && rm -rf /var/lib/apt/lists/*"}).
		WithServiceBinding("mysql", mysql).
		WithServiceBinding("temporal", temporal).
		WithServiceBinding("ambox", ambox).
		WithServiceBinding("redis", redis).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/storage", volumes.storage).
		WithMountedCache("/runtime/coverage", volumes.coverage).
		WithEnvVariable("GOCOVERDIR", "/runtime/coverage/raw").
		WithEnvVariable("AWS_ACCESS_KEY_ID", "minio").
		WithEnvVariable("AWS_SECRET_ACCESS_KEY", "minio123").
		WithEnvVariable("S3_BUCKET", "sips").
		WithFile("/usr/local/bin/enduro", enduroBin, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithFile("/usr/local/bin/weed", weedBin, dagger.ContainerWithFileOpts{
			Permissions: 0o755,
		}).
		WithNewFile("/etc/enduro/enduro.ambox.toml", enduroObjectStorageConfig(objectStorageProviderSeaweedFS, "http://127.0.0.1:8333"), dagger.ContainerWithNewFileOpts{
			Permissions: 0o644,
		}).
		WithNewFile("/etc/seaweedfs/notification.toml", seaweedFSNotificationConfig("http://127.0.0.1:7480/seaweedfs/events"), dagger.ContainerWithNewFileOpts{
			Permissions: 0o644,
		}).
		WithNewFile("/etc/enduro/ssh/sftp_key", e2eSFTPPrivateKey, dagger.ContainerWithNewFileOpts{
			Permissions: 0o600,
		}).
		WithNewFile("/usr/local/bin/enduro-e2e-entrypoint", seaweedFSEnduroEntrypointScript, dagger.ContainerWithNewFileOpts{
			Permissions: 0o755,
		}).
		WithExposedPort(9000).
		WithExposedPort(9001).
		WithExposedPort(7480).
		WithExposedPort(8333).
		WithEntrypoint([]string{"/usr/local/bin/enduro-e2e-entrypoint"}).
		AsService(dagger.ContainerAsServiceOpts{
			UseEntrypoint: true,
		})
}

func (m *EnduroE2E) enduroBase(source *dagger.Directory, volumes runtimeVolumes) *dagger.Container {
	frontendAssets := m.frontendAssets(source)

	return dag.Container().
		From(goImage).
		WithDirectory("/src", source, dagger.ContainerWithDirectoryOpts{
			Exclude: []string{
				".git",
				"build",
				"dist",
				"frontend/.nuxt",
				"frontend/.output",
				"frontend/node_modules",
				"frontend/coverage",
				"hack/dagger/runtime",
				"hack/minio-data",
				"hack/seaweedfs-data",
				"ui/node_modules",
			},
		}).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("enduro-e2e-go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("enduro-e2e-go-build")).
		WithMountedCache("/runtime/batch", volumes.batch).
		WithMountedCache("/runtime/watched", volumes.watched).
		WithMountedCache("/runtime/completed", volumes.completed).
		WithMountedCache("/runtime/transfers", volumes.transfers).
		WithMountedCache("/runtime/processing", volumes.processing).
		WithMountedCache("/runtime/storage", volumes.storage).
		WithDirectory("/src/frontend/.output/public", frontendAssets).
		WithExec([]string{"sh", "-ceu", "apt-get update && apt-get install -y --no-install-recommends ca-certificates unzip p7zip-full && rm -rf /var/lib/apt/lists/*"}).
		WithExec([]string{"go", "build", "-cover", "-covermode=atomic", "-coverpkg=./...", "-trimpath", "-o", "build/enduro", "."})
}

func (m *EnduroE2E) frontendAssets(source *dagger.Directory) *dagger.Directory {
	return dag.Container().
		From(nodeImage).
		WithDirectory("/src/frontend", source.Directory("frontend"), dagger.ContainerWithDirectoryOpts{
			Exclude: []string{
				".nuxt",
				".output",
				"coverage",
				"node_modules",
			},
		}).
		WithWorkdir("/src/frontend").
		WithMountedCache("/root/.npm", dag.CacheVolume("enduro-e2e-npm-cache")).
		WithExec([]string{"npm", "ci"}).
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
				"frontend/node_modules",
				"frontend/coverage",
				"hack/dagger/runtime",
				"hack/minio-data",
				"hack/seaweedfs-data",
				"ui/node_modules",
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
				"frontend/node_modules",
				"frontend/coverage",
				"hack/dagger/runtime",
				"hack/minio-data",
				"hack/seaweedfs-data",
				"ui/node_modules",
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
legacyListen = ""
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
receiptPath = ""
disabled = true

[metadata]
processNameMetadata = false

[worker]
heartbeatThrottleInterval = "1m"
maxConcurrentWorkflowsExecutionsSize = 10
maxConcurrentSessionExecutionSize = 4

[workflow]
activityHeartbeatTimeout = "30s"
`)
}

func enduroObjectStorageConfig(provider objectStorageProvider, endpoint string) string {
	var webhookConfig string
	eventFormat := "minio"
	redisList := "minio-events"
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
legacyListen = ""
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
	return provider == objectStorageProviderMinioLegacy || provider == objectStorageProviderMinioLatest
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

const enduroEntrypointScript = `#!/bin/sh
set -eu

mkdir -p /etc/enduro/ssh
for i in $(seq 1 90); do
	if ssh-keyscan -p 64022 ambox > /etc/enduro/ssh/known_hosts 2>/dev/null; then
		exec /usr/local/bin/enduro --config /etc/enduro/enduro.ambox.toml
	fi
	sleep 2
done

echo "timed out waiting for ambox SFTP host key" >&2
exit 1
`

const seaweedFSEnduroEntrypointScript = `#!/bin/sh
set -eu

mkdir -p /etc/enduro/ssh /tmp/seaweedfs
/usr/local/bin/weed mini \
	-dir=/tmp/seaweedfs \
	-ip.bind=0.0.0.0 \
	-bucket=sips \
	-master.volumeSizeLimitMB=1024 \
	-webdav=false \
	-admin.ui=false \
	-s3.port.iceberg=0 &

for i in $(seq 1 90); do
	if curl -fsS http://127.0.0.1:8888/ >/dev/null 2>&1; then
		break
	fi
	if [ "$i" = "90" ]; then
		echo "timed out waiting for SeaweedFS filer" >&2
		exit 1
	fi
	sleep 1
done

for i in $(seq 1 90); do
	if ssh-keyscan -p 64022 ambox > /etc/enduro/ssh/known_hosts 2>/dev/null; then
		exec /usr/local/bin/enduro --config /etc/enduro/enduro.ambox.toml
	fi
	sleep 2
done

echo "timed out waiting for ambox SFTP host key" >&2
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
