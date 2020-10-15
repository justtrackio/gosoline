module github.com/applike/gosoline

require (
	github.com/DATA-DOG/go-sqlmock v1.4.1
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/Masterminds/squirrel v1.2.0
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/alicebob/gopher-json v0.0.0-20180125190556-5a6b3ba71ee6 // indirect
	github.com/alicebob/miniredis v2.4.6+incompatible
	github.com/apache/thrift v0.13.0 // indirect
	github.com/aws/aws-lambda-go v1.13.2
	github.com/aws/aws-sdk-go v1.19.37
	github.com/aws/aws-xray-sdk-go v1.1.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/certifi/gocertifi v0.0.0-20180118203423-deb3ae2ef261 // indirect
	github.com/containerd/continuity v0.0.0-20190426062206-aaeac12a7ffc // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190313032549-041949b8d268 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/elastic/go-elasticsearch/v6 v6.8.3-0.20190714143207-256a620be07d
	github.com/elastic/go-elasticsearch/v7 v7.2.1-0.20190714143206-f1e755531ff4
	github.com/elliotchance/redismock v1.4.0
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/fatih/color v1.7.0
	github.com/getsentry/raven-go v0.2.0
	github.com/gin-contrib/cors v0.0.0-20190301062745-f9e10995c85a
	github.com/gin-gonic/gin v1.4.0
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/go-redis/redis v6.15.1+incompatible
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d // indirect
	github.com/golang-migrate/migrate/v4 v4.2.5
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/google/go-querystring v1.0.0
	github.com/google/uuid v1.1.1
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/jackc/pgx/v4 v4.8.1
	github.com/jeremywohl/flatten v0.0.0-20190921043622-d936035e55cf
	github.com/jinzhu/gorm v1.9.2
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a
	github.com/jinzhu/now v0.0.0-20181116074157-8ec929ed50c3 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/jonboulle/clockwork v0.1.0
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/karlseguin/ccache v0.0.0-20181227155450-692cd618b264
	github.com/karlseguin/expect v1.0.1 // indirect
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lib/pq v1.3.0
	github.com/mitchellh/mapstructure v1.2.2
	github.com/myesui/uuid v1.0.0 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/ory/dockertest v3.3.5+incompatible
	github.com/ory/ladon v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/sha1sum/aws_signing_client v0.0.0-20170514202702-9088e4c7b34b
	github.com/sirupsen/logrus v1.5.0 // indirect
	github.com/spf13/cast v1.3.0
	github.com/stretchr/objx v0.2.0
	github.com/stretchr/testify v1.5.1
	github.com/thoas/go-funk v0.0.0-20181020164546-fbae87fb5b5c
	github.com/tidwall/gjson v1.3.0
	github.com/twinj/uuid v1.0.0
	github.com/twitchscience/kinsumer v0.0.0-20190125174422-b6682f9326f7
	github.com/ugorji/go v1.1.7 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/wsxiaoys/terminal v0.0.0-20160513160801-0940f3fc43a0 // indirect
	github.com/xitongsys/parquet-go v1.4.0
	github.com/xitongsys/parquet-go-source v0.0.0-20191104003508-ecfa341356a6
	github.com/yuin/gopher-lua v0.0.0-20190206043414-8bfc7677f583 // indirect
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4
	google.golang.org/api v0.5.0
	gopkg.in/go-playground/validator.v8 v8.18.2
	gopkg.in/go-playground/validator.v9 v9.30.0
	gopkg.in/karlseguin/expect.v1 v1.0.1 // indirect
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/stretchr/testify.v1 v1.2.2 // indirect
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gopkg.in/yaml.v2 v2.2.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20191120175047-4206685974f2
)

go 1.13
