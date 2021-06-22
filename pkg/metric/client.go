package metric

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"net/http"
	"sync"
	"time"
)

var cwcl = struct {
	sync.Mutex
	client      cloudwatchiface.CloudWatchAPI
	initialized bool
}{}

func ProvideCloudWatchClient(config cfg.Config) cloudwatchiface.CloudWatchAPI {
	cwcl.Lock()
	defer cwcl.Unlock()

	if cwcl.initialized {
		return cwcl.client
	}

	endpoint := config.GetString("aws_cloudwatch_endpoint")
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

	client := cloudwatch.New(sess)

	cwcl.client = client
	cwcl.initialized = true

	return cwcl.client
}
