package cloud

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	LogDebug                    = "debug"
	LogDebugWithEventStreamBody = "debug_event_stream_body"
	LogDebugWithHTTPBody        = "debug_http"
	LogDebugWithRequestErrors   = "debug_request_errors"
	LogDebugWithRequestRetries  = "debug_request_retries"
	LogDebugWithSigning         = "debug_signing"
	LogOff                      = "off"
)

func LogLevelStringToAwsLevel(level string) aws.LogLevelType {
	switch level {
	case LogDebug:
		return aws.LogDebug
	case LogDebugWithEventStreamBody:
		return aws.LogDebugWithEventStreamBody
	case LogDebugWithHTTPBody:
		return aws.LogDebugWithHTTPBody
	case LogDebugWithRequestErrors:
		return aws.LogDebugWithRequestErrors
	case LogDebugWithRequestRetries:
		return aws.LogDebugWithRequestRetries
	case LogDebugWithSigning:
		return aws.LogDebugWithSigning
	case LogOff:
		return aws.LogOff
	}

	return aws.LogOff
}

type ClientSettings struct {
	MaxRetries  int           `cfg:"max_retries" default:"10"`
	HttpTimeout time.Duration `cfg:"http_timeout" default:"1m"`
	LogLevel    string        `cfg:"log_level" default:"off"`
}

func GetAwsConfig(config cfg.Config, logger log.Logger, service string, settings *ClientSettings) *aws.Config {
	srvCfgKey := fmt.Sprintf("aws_%s_endpoint", service)

	endpoint := config.GetString(srvCfgKey)
	maxRetries := settings.MaxRetries
	logLevel := LogLevelStringToAwsLevel(settings.LogLevel)
	httpTimeout := time.Minute

	if settings.HttpTimeout > 0 {
		httpTimeout = settings.HttpTimeout
	}

	return &aws.Config{
		Credentials:                   gosoAws.GetDefaultCredentials(),
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(endpoints.EuCentral1RegionID),
		Endpoint:                      aws.String(endpoint),
		MaxRetries:                    aws.Int(maxRetries),
		HTTPClient: &http.Client{
			Timeout: httpTimeout,
		},
		Logger:   PrefixedLogger(logger, service),
		LogLevel: aws.LogLevel(logLevel),
	}
}

/* Configuration Template for AWS Clients */
var ConfigTemplate = aws.Config{
	CredentialsChainVerboseErrors: aws.Bool(true),
	Region:                        aws.String(endpoints.EuCentral1RegionID),
	// LogLevel: aws.LogLevel(aws.LogDebug | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors | aws.LogDebugWithHTTPBody),
	HTTPClient: &http.Client{
		Timeout: 1 * time.Minute,
	},
}

/* DynamoDB client */
var ddbcl = struct {
	sync.Mutex
	client map[string]dynamodbiface.DynamoDBAPI
}{}

func GetDynamoDbClient(config cfg.Config, logger log.Logger) DynamoDBAPI {
	ddbcl.Lock()
	defer ddbcl.Unlock()

	if ddbcl.client == nil {
		ddbcl.client = map[string]dynamodbiface.DynamoDBAPI{}
	}

	endpoint := config.GetString("aws_dynamoDb_endpoint")

	if ddbcl.client[endpoint] != nil {
		return ddbcl.client[endpoint]
	}

	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_dynamo_db"))

	sess := session.Must(session.NewSession(awsConfig.WithEndpoint(endpoint)))

	client := dynamodb.New(sess)

	ddbcl.client[endpoint] = client

	return ddbcl.client[endpoint]
}

/* Kinesis client */
var kcl = struct {
	sync.Mutex
	client      kinesisiface.KinesisAPI
	initialized bool
}{}

func GetKinesisClient(config cfg.Config, logger log.Logger) KinesisAPI {
	kcl.Lock()
	defer kcl.Unlock()

	if kcl.initialized {
		return kcl.client
	}

	endpoint := config.GetString("aws_kinesis_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_kinesis"))

	sess := session.Must(session.NewSession(&awsConfig))

	client := kinesis.New(sess)

	kcl.client = client
	kcl.initialized = true

	return kcl.client
}

func PrefixedLogger(logger log.Logger, service string) aws.LoggerFunc {
	return func(args ...interface{}) {
		logger.WithFields(log.Fields{
			"aws_service": service,
		}).Info(fmt.Sprint(args...))
	}
}
