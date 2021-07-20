package cloud

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/applike/gosoline/pkg/cfg"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling/applicationautoscalingiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
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

func GetAwsConfig(config cfg.Config, logger mon.Logger, service string, settings *ClientSettings) *aws.Config {
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

func GetDynamoDbClient(config cfg.Config, logger mon.Logger) DynamoDBAPI {
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

/* ApplicationAutoscaling client */
var aacl = struct {
	sync.Mutex
	client      applicationautoscalingiface.ApplicationAutoScalingAPI
	initialized bool
}{}

func GetApplicationAutoScalingClient(config cfg.Config, logger mon.Logger) ApplicationAutoScalingAPI {
	aacl.Lock()
	defer aacl.Unlock()

	if aacl.initialized {
		return aacl.client
	}

	endpoint := config.GetString("aws_application_autoscaling_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_application_autoscaling"))

	sess := session.Must(session.NewSession(&awsConfig))

	client := applicationautoscaling.New(sess)

	aacl.client = client
	aacl.initialized = true

	return aacl.client
}

/* EC2 Client */
var ec2cl = struct {
	sync.Mutex
	client      ec2iface.EC2API
	initialized bool
}{}

func GetEc2Client(logger mon.Logger) EC2API {
	ec2cl.Lock()
	defer ec2cl.Unlock()

	if ec2cl.initialized {
		return ec2cl.client
	}

	awsConfig := ConfigTemplate
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_ec2"))

	sess := session.Must(session.NewSession(&awsConfig))

	ec2cl.client = ec2.New(sess)
	ec2cl.initialized = true

	return ec2cl.client
}

/* ECS Client */
var ecscl = struct {
	sync.Mutex
	client      ecsiface.ECSAPI
	initialized bool
}{}

func GetEcsClient(logger mon.Logger) ECSAPI {
	ecscl.Lock()
	defer ecscl.Unlock()

	if ecscl.initialized {
		return ecscl.client
	}

	awsConfig := ConfigTemplate
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_ecs"))

	sess := session.Must(session.NewSession(&awsConfig))

	ecscl.client = ecs.New(sess)
	ecscl.initialized = true

	return ecscl.client
}

/* Kinesis client */
var kcl = struct {
	sync.Mutex
	client      kinesisiface.KinesisAPI
	initialized bool
}{}

func GetKinesisClient(config cfg.Config, logger mon.Logger) KinesisAPI {
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

/* Rds Client */
var rdsClient = struct {
	sync.Mutex
	client      rdsiface.RDSAPI
	initialized bool
}{}

func GetRdsClient(config cfg.Config, logger mon.Logger) RDSAPI {
	rdsClient.Lock()
	defer rdsClient.Unlock()

	if rdsClient.initialized {
		return rdsClient.client
	}

	endpoint := config.GetString("aws_rds_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_rds"))
	sess := session.Must(session.NewSession(&awsConfig))

	rdsClient.client = rds.New(sess)
	rdsClient.initialized = true

	return rdsClient.client
}

/* ServiceDiscovery Client */
var sdcl = struct {
	sync.Mutex
	client      servicediscoveryiface.ServiceDiscoveryAPI
	initialized bool
}{}

func GetServiceDiscoveryClient(logger mon.Logger, endpoint string) ServiceDiscoveryAPI {
	sdcl.Lock()
	defer sdcl.Unlock()

	if sdcl.initialized {
		return sdcl.client
	}

	awsConfig := ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_service_discovery"))
	sess := session.Must(session.NewSession(&awsConfig))

	sdcl.client = servicediscovery.New(sess)
	sdcl.initialized = true

	return sdcl.client
}

/* SimpleSystemsManager Client */
var ssmClient = struct {
	sync.Mutex
	client      ssmiface.SSMAPI
	initialized bool
}{}

func GetSystemsManagerClient(config cfg.Config, logger mon.Logger) SSMAPI {
	ssmClient.Lock()
	defer ssmClient.Unlock()

	if ssmClient.initialized {
		return ssmClient.client
	}

	endpoint := config.GetString("aws_ssm_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(PrefixedLogger(logger, "aws_systems_manager"))
	sess := session.Must(session.NewSession(&awsConfig))

	ssmClient.client = ssm.New(sess)
	ssmClient.initialized = true

	return ssmClient.client
}

func PrefixedLogger(logger mon.Logger, service string) aws.LoggerFunc {
	return func(args ...interface{}) {
		logger.WithFields(mon.Fields{
			"aws_service": service,
		}).Info(fmt.Sprint(args...))
	}
}
