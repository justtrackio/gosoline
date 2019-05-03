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

	endpoint := config.GetString("aws_s3_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(endpoints.EuCentral1RegionID),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)

	sess := session.Must(session.NewSession(awsConfig.WithEndpoint(endpoint)))

	client := s3.New(sess)

	s3cl.client = client
	s3cl.initialized = true

	return s3cl.client
}
