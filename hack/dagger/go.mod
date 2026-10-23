module dagger/enduro-e-2-e

// Keep the go directive compatible with Dagger's bundled Go toolchain.
go 1.26.5

toolchain go1.26.10

require (
	github.com/Khan/genqlient v0.8.1
	github.com/dagger/otel-go v1.43.0
	github.com/dagger/querybuilder v0.0.0-20260402040506-574a5e81cb59
	github.com/vektah/gqlparser/v2 v2.5.64
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/sdk v1.46.0
	go.opentelemetry.io/otel/trace v1.46.0
)

require (
	github.com/99designs/gqlgen v0.17.98 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.31.0 // indirect
	github.com/sosodev/duration v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.23.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.46.0 // indirect
	go.opentelemetry.io/otel/log v1.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/sdk/log v1.47.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.46.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.1 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20261019180522-ce908d29299b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261019180522-ce908d29299b // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

// dagger/otel-go still uses log.Value and log.KeyValue, removed in newer log APIs.
// Keep the logging API, SDK, and exporters on the compatible release together.
// Keep the core OpenTelemetry modules at v1.46.0: v1.47.0 introduces an import
// cycle with log v0.19.0. grpc-gateway stays at v2.31.0 because v2.32.0 requires
// OpenTelemetry v1.47.0. Logging v0.19.0 fixes GO-2026-4985; GO-2026-6508 and
// GO-2026-6615 remain until Dagger supports the newer logging APIs.
// Upstream tracking: https://github.com/dagger/dagger/issues/13040
replace go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc => go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0

replace go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp => go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.0

replace go.opentelemetry.io/otel/log => go.opentelemetry.io/otel/log v0.19.0

replace go.opentelemetry.io/otel/sdk/log => go.opentelemetry.io/otel/sdk/log v0.19.0
