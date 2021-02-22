module github.com/artefactual-labs/enduro

go 1.15

require (
	github.com/GeertJohan/go.rice v1.0.1-0.20190430230923-c880e3cd4dd8
	github.com/alicebob/miniredis/v2 v2.13.1
	github.com/anmitsu/go-shlex v0.0.0-20200502080107-070676123096 // indirect
	github.com/atrox/go-migrate-rice v1.0.1
	github.com/aws/aws-sdk-go v1.35.1
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/daaku/go.zipexe v1.0.1 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/frankban/quicktest v1.5.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0
	github.com/go-redis/redis/v7 v7.4.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang-migrate/migrate/v4 v4.13.0
	github.com/golang/mock v1.4.4
	github.com/golang/snappy v0.0.2 // indirect
	github.com/google/uuid v1.1.2
	github.com/gorilla/schema v1.2.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/otiai10/copy v1.2.0
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.14.0 // indirect
	github.com/prometheus/procfs v0.2.0 // indirect
	github.com/radovskyb/watcher v1.0.7
	github.com/spf13/afero v1.4.0
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/uber-go/tally v3.3.17+incompatible
	github.com/uber/tchannel-go v1.20.1 // indirect
	github.com/ulikunitz/xz v0.5.8 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.uber.org/cadence v0.14.1
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/yarpc v1.52.0
	go.uber.org/zap v1.16.0
	goa.design/goa/v3 v3.2.4
	goa.design/plugins/v3 v3.2.4
	gocloud.dev v0.20.0
	golang.org/x/net v0.0.0-20200930145003-4acb6c075d10 // indirect
	golang.org/x/sync v0.0.0-20200930132711-30421366ff76
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	golang.org/x/tools v0.0.0-20201002161817-08f19738fac6 // indirect
	google.golang.org/genproto v0.0.0-20201002142447-3860012362da // indirect
	google.golang.org/grpc v1.32.0 // indirect
	gopkg.in/ini.v1 v1.61.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	gotest.tools/v3 v3.0.2
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
)

// "go.uber.org/cadence" requires it but "go mod" selects "v0.12.0".
// I suspect the problem is in that Thrift tags are not using the "v" prefix.
replace github.com/apache/thrift => github.com/apache/thrift v0.0.0-20161221203622-b2a4d4ae21c7
