module github.com/justtrackio/gosoline

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Masterminds/squirrel v1.5.2
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/aws/aws-lambda-go v1.29.0
	github.com/aws/aws-sdk-go v1.43.40
	github.com/aws/aws-sdk-go-v2 v1.16.2
	github.com/aws/aws-sdk-go-v2/config v1.15.3
	github.com/aws/aws-sdk-go-v2/credentials v1.11.2
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.8.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.4.4
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.18.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.15.3
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.35.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.18.5
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.15.3
	github.com/aws/aws-sdk-go-v2/service/rds v1.18.4
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.13.4
	github.com/aws/aws-sdk-go-v2/service/s3 v1.26.5
	github.com/aws/aws-sdk-go-v2/service/sns v1.17.4
	github.com/aws/aws-sdk-go-v2/service/sqs v1.18.3
	github.com/aws/aws-sdk-go-v2/service/ssm v1.24.1
	github.com/aws/aws-xray-sdk-go v1.7.0
	github.com/aws/smithy-go v1.11.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elastic/go-elasticsearch/v7 v7.2.1-0.20190714143206-f1e755531ff4
	github.com/elliotchance/redismock/v8 v8.11.0
	github.com/fatih/color v1.13.0
	github.com/getsentry/sentry-go v0.13.0
	github.com/gin-contrib/cors v0.0.0-20190301062745-f9e10995c85a
	github.com/gin-contrib/gzip v0.0.5
	github.com/gin-contrib/location v0.0.2
	github.com/gin-gonic/gin v1.7.7
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a
	github.com/go-playground/validator/v10 v10.10.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-resty/resty/v2 v2.7.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-migrate/migrate/v4 v4.15.1
	github.com/google/go-querystring v1.1.0
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/iancoleman/strcase v0.2.0
	github.com/imdario/mergo v0.3.12
	github.com/jackc/pgx/v4 v4.15.0
	github.com/jeremywohl/flatten v0.0.0-20190921043622-d936035e55cf
	github.com/jinzhu/gorm v1.9.16
	github.com/jinzhu/inflection v1.0.0
	github.com/jmoiron/sqlx v1.3.4
	github.com/karlseguin/ccache v0.0.0-20181227155450-692cd618b264
	github.com/lib/pq v1.10.4
	github.com/mitchellh/mapstructure v1.4.3
	github.com/ory/dockertest/v3 v3.8.1
	github.com/ory/ladon v1.2.0
	github.com/oschwald/geoip2-golang v1.7.0
	github.com/pkg/errors v0.9.1
	github.com/sha1sum/aws_signing_client v0.0.0-20170514202702-9088e4c7b34b
	github.com/spf13/cast v1.4.1
	github.com/stretchr/testify v1.7.1
	github.com/thoas/go-funk v0.0.0-20181020164546-fbae87fb5b5c
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/xitongsys/parquet-go v1.4.0
	github.com/xitongsys/parquet-go-source v0.0.0-20191104003508-ecfa341356a6
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220328115105-d36c6a25d886
	google.golang.org/api v0.74.0
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require (
	cloud.google.com/go/compute v1.5.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/DataDog/zstd v1.4.8 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/apache/thrift v0.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.3 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/continuity v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/docker/cli v20.10.11+incompatible // indirect
	github.com/docker/docker v20.10.9+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gomodule/redigo v1.7.1-0.20190724094224-574c33c3df38 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/gax-go/v2 v2.2.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.11.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.2.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.10.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/karlseguin/expect v1.0.8 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/runc v1.0.2 // indirect
	github.com/oschwald/maxminddb-golang v1.9.0 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.34.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yuin/gopher-lua v0.0.0-20200816102855-ee81675732da // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/net v0.0.0-20220325170049-de3da57026de // indirect
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220324131243-acbaeb5b85eb // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

go 1.18
