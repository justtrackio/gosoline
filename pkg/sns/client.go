package sns

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"sync"
)

//go:generate mockery -name Client
type Client interface {
	snsiface.SNSAPI
}

type client struct {
	sync.Mutex
	client      Client
	initialized bool
}

var c = client{}

func GetClient(config cfg.Config, logger mon.Logger) Client {
	c.Lock()
	defer c.Unlock()

	if c.initialized {
		return c.client
	}

	c.client = buildClient(config, logger)
	c.initialized = true

	return c.client
}

func buildClient(config cfg.Config, logger mon.Logger) *sns.SNS {
	endpoint := config.GetString("aws_sns_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := cloud.ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(cloud.PrefixedLogger(logger, "sns"))

	sess := session.Must(session.NewSession(awsConfig))

	return sns.New(sess)
}
