package sns

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"sync"
)

//go:generate mockery -name Client
type Client interface {
	snsiface.SNSAPI
}

var c = struct {
	sync.Mutex
	instance *sns.SNS
}{}

func ProvideClient(config cfg.Config, logger log.Logger, settings *Settings) *sns.SNS {
	c.Lock()
	defer c.Unlock()

	if c.instance != nil {
		return c.instance
	}

	c.instance = NewClient(config, logger, settings)

	return c.instance
}

func NewClient(config cfg.Config, logger log.Logger, settings *Settings) *sns.SNS {
	if settings.Backoff.Enabled {
		settings.Client.MaxRetries = 0
	}

	awsConfig := cloud.GetAwsConfig(config, logger, "sns", &settings.Client)
	sess := session.Must(session.NewSession(awsConfig))

	return sns.New(sess)
}
