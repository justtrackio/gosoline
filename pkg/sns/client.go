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

func GetClient(config cfg.Config, logger mon.Logger, settings *cloud.ClientSettings) Client {
	c.Lock()
	defer c.Unlock()

	if c.initialized {
		return c.client
	}

	c.client = buildClient(config, logger, settings)
	c.initialized = true

	return c.client
}

func buildClient(config cfg.Config, logger mon.Logger, settings *cloud.ClientSettings) *sns.SNS {
	awsConfig := cloud.GetAwsConfig(config, logger, "sns", settings)
	sess := session.Must(session.NewSession(awsConfig))

	return sns.New(sess)
}
