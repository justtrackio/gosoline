package resources

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"sync"
)

var rgtClient = struct {
	sync.Mutex
	client      resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	initialized bool
}{}

func GetClient(config cfg.Config, logger log.Logger) resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI {
	rgtClient.Lock()
	defer rgtClient.Unlock()

	if rgtClient.initialized {
		return rgtClient.client
	}

	endpoint := config.GetString("aws_rgt_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := cloud.ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(cloud.PrefixedLogger(logger, "aws_resources_manager"))
	sess := session.Must(session.NewSession(&awsConfig))

	rgtClient.client = resourcegroupstaggingapi.New(sess)
	rgtClient.initialized = true

	return rgtClient.client
}
