module github.com/artefactual-labs/enduro

go 1.16

require (
	github.com/alicebob/miniredis/v2 v2.15.0
	github.com/anmitsu/go-shlex v0.0.0-20200502080107-070676123096 // indirect
	github.com/aws/aws-sdk-go v1.40.2
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/go-redis/redis/v8 v8.10.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-migrate/migrate/v4 v4.14.2-0.20201125065321-a53e6fc42574
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/schema v1.2.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jmoiron/sqlx v1.3.4
	github.com/johejo/golang-migrate-extra v0.0.0-20210217013041-51a992e50d16
	github.com/jonboulle/clockwork v0.2.2
	github.com/kr/pretty v0.2.1 // indirect
	github.com/mholt/archiver/v3 v3.5.0
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/otiai10/copy v1.6.0
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/radovskyb/watcher v1.0.7
	github.com/spf13/afero v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/twmb/murmur3 v1.1.5 // indirect
	github.com/uber-go/tally v3.4.1+incompatible
	github.com/uber/tchannel-go v1.20.1 // indirect
	github.com/ulikunitz/xz v0.5.8 // indirect
	go.uber.org/cadence v0.17.0
	go.uber.org/yarpc v1.54.2
	go.uber.org/zap v1.18.1
	goa.design/goa/v3 v3.4.3
	goa.design/plugins/v3 v3.4.3
	gocloud.dev v0.23.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gotest.tools/v3 v3.0.3
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
)

// "go.uber.org/cadence" requires it but "go mod" selects "v0.12.0".
// I suspect the problem is in that Thrift tags are not using the "v" prefix.
replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7
