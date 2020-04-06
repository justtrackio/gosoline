package blob

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"net/http"
	"sync"
	"time"
)

var s3cl = struct {
	sync.Mutex
	client      s3iface.S3API
	initialized bool
}{}

func ProvideS3Client(config cfg.Config) s3iface.S3API {
	s3cl.Lock()
	defer s3cl.Unlock()

	if s3cl.initialized {
		return s3cl.client
	}

	awsConfig := GetS3ClientConfig(config)
	sess := session.Must(session.NewSession(awsConfig))

	client := s3.New(sess)

	s3cl.client = client
	s3cl.initialized = true

	return s3cl.client
}

func GetS3ClientConfig(config cfg.Config) *aws.Config {
	endpoint := config.GetString("aws_s3_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")
	timeout := 1 * time.Minute
	if config.IsSet("aws_s3_timeout") {
		timeout = config.GetDuration("aws_s3_timeout")
	}

	return &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Endpoint:                      aws.String(endpoint),
		Region:                        aws.String(endpoints.EuCentral1RegionID),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		MaxRetries: aws.Int(maxRetries),
	}
}
