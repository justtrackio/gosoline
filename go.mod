module github.com/justtrackio/gosoline

require (
	github.com/DATA-DOG/go-sqlmock v1.4.1
	github.com/Masterminds/squirrel v1.2.0
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/alicebob/miniredis v2.4.6+incompatible
	github.com/aws/aws-lambda-go v1.13.2
	github.com/aws/aws-sdk-go v1.38.7
	github.com/aws/aws-sdk-go-v2 v1.9.1
	github.com/aws/aws-sdk-go-v2/config v1.1.3
	github.com/aws/aws-sdk-go-v2/credentials v1.1.3
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.1.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.2.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.2.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.4.2
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.18.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.9.1
	github.com/aws/aws-sdk-go-v2/service/rds v1.9.0
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.5.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.16.0
	github.com/aws/aws-sdk-go-v2/service/sns v1.7.1
	github.com/aws/aws-sdk-go-v2/service/sqs v1.7.1
	github.com/aws/aws-sdk-go-v2/service/ssm v1.11.0
	github.com/aws/aws-xray-sdk-go v1.1.0
	github.com/aws/smithy-go v1.8.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elastic/go-elasticsearch/v7 v7.2.1-0.20190714143206-f1e755531ff4
	github.com/elliotchance/redismock/v8 v8.6.1
	github.com/fatih/color v1.7.0
	github.com/getsentry/sentry-go v0.11.0
	github.com/gin-contrib/cors v0.0.0-20190301062745-f9e10995c85a
	github.com/gin-contrib/gzip v0.0.3
	github.com/gin-gonic/gin v1.7.7
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a
	github.com/go-playground/validator/v10 v10.4.1
	github.com/go-redis/redis/v8 v8.8.0
	github.com/go-resty/resty/v2 v2.6.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-migrate/migrate/v4 v4.2.5
	github.com/google/go-querystring v1.0.0
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/iancoleman/strcase v0.1.3
	github.com/imdario/mergo v0.3.12
	github.com/jackc/pgx/v4 v4.8.1
	github.com/jeremywohl/flatten v0.0.0-20190921043622-d936035e55cf
	github.com/jinzhu/gorm v1.9.16
	github.com/jinzhu/inflection v1.0.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/jonboulle/clockwork v0.1.0
	github.com/karlseguin/ccache v0.0.0-20181227155450-692cd618b264
	github.com/lib/pq v1.3.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/ory/dockertest/v3 v3.7.0
	github.com/ory/ladon v1.0.1
	github.com/oschwald/geoip2-golang v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/sha1sum/aws_signing_client v0.0.0-20170514202702-9088e4c7b34b
	github.com/spf13/cast v1.3.0
	github.com/stretchr/objx v0.2.0
	github.com/stretchr/testify v1.7.0
	github.com/thoas/go-funk v0.0.0-20181020164546-fbae87fb5b5c
	github.com/twitchscience/kinsumer v0.0.0-20201111182439-cd685b6b5f68
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/xitongsys/parquet-go v1.6.2
	github.com/xitongsys/parquet-go-source v0.0.0-20200817004010-026bad9b25d0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	google.golang.org/api v0.20.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.2.3 // indirect
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/karlseguin/expect v1.0.8 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	golang.org/x/text v0.3.6 // indirect
)

go 1.17
