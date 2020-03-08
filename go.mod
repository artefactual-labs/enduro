module github.com/artefactual-labs/enduro

go 1.14

require (
	github.com/GeertJohan/go.rice v1.0.1-0.20190430230923-c880e3cd4dd8
	github.com/apache/thrift v0.13.0 // indirect
	github.com/atrox/go-migrate-rice v1.0.1
	github.com/aws/aws-sdk-go v1.29.19
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/daaku/go.zipexe v1.0.1 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/frankban/quicktest v1.5.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-redis/redis/v7 v7.2.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/gogo/status v1.1.0 // indirect
	github.com/golang-migrate/migrate/v4 v4.9.1
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.4.1
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/google/uuid v1.1.1
	github.com/google/wire v0.4.0 // indirect
	github.com/gorilla/schema v1.1.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pierrec/lz4 v2.4.1+incompatible // indirect
	github.com/prometheus/client_golang v1.5.0
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/radovskyb/watcher v1.0.7
	github.com/samuel/go-thrift v0.0.0-20191111193933-5165175b40af // indirect
	github.com/spf13/afero v1.2.2
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber-go/tally v3.3.15+incompatible
	github.com/uber/tchannel-go v1.17.0 // indirect
	github.com/ulikunitz/xz v0.5.7 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/cadence v0.11.2
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/net/metrics v1.3.0 // indirect
	go.uber.org/thriftrw v1.22.0 // indirect
	go.uber.org/yarpc v1.44.0
	go.uber.org/zap v1.14.0
	goa.design/goa v2.0.10+incompatible
	goa.design/goa/v3 v3.0.10
	goa.design/plugins/v3 v3.0.10
	gocloud.dev v0.19.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20200308013534-11ec41452d41 // indirect
	google.golang.org/genproto v0.0.0-20200306153348-d950eab6f860 // indirect
	google.golang.org/grpc v1.27.1 // indirect
	gopkg.in/ini.v1 v1.54.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
	gotest.tools/v3 v3.0.2
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
)

// "go.uber.org/cadence" requires it but "go mod" selects "v0.12.0".
// I suspect the problem is in that Thrift tags are not using the "v" prefix.
replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7

// v1.5.0 not released yet!
replace github.com/go-sql-driver/mysql => github.com/go-sql-driver/mysql v1.4.1-0.20191001060945-14bb9c0fc20f
