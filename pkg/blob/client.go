package blob

import (
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

var s3Clients = struct {
	sync.Mutex
	clients map[string]s3iface.S3API
}{
	clients: map[string]s3iface.S3API{},
}

func ProvideS3Client(config cfg.Config) s3iface.S3API {
	currentEndpoint := config.GetString("aws_s3_endpoint")
	s3Clients.Lock()
	defer s3Clients.Unlock()

	if client, ok := s3Clients.clients[currentEndpoint]; ok {
		return client
	}

	awsConfig := GetS3ClientConfig(config)
	sess := session.Must(session.NewSession(awsConfig))

	client := s3.New(sess)

	s3Clients.clients[currentEndpoint] = client

	return client
}

func GetS3ClientConfig(config cfg.Config) *aws.Config {
	endpoint := config.GetString("aws_s3_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")
	s3ForcePathStyle := config.GetBool("aws_s3_forcePathStyle", false)

	return &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Endpoint:                      aws.String(endpoint),
		Region:                        aws.String(endpoints.EuCentral1RegionID),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
		MaxRetries:       aws.Int(maxRetries),
		S3ForcePathStyle: aws.Bool(s3ForcePathStyle),
	}
}
