package ses

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"sync"
)

//go:generate mockery -name Client
type Client interface {
	sesiface.SESAPI
}

var c = struct {
	sync.Mutex
	instance *ses.SES
}{}

func ProvideClient(config cfg.Config, logger log.Logger, settings *Settings) *ses.SES {
	c.Lock()
	defer c.Unlock()

	if c.instance != nil {
		return c.instance
	}

	c.instance = NewClient(config, logger, settings)

	return c.instance
}

func NewClient(config cfg.Config, logger log.Logger, settings *Settings) *ses.SES {
	if settings.Backoff.Enabled {
		settings.Client.MaxRetries = 0
	}

	awsConfig := cloud.GetAwsConfig(config, logger, "ses", &settings.Client)
	sess := session.Must(session.NewSession(awsConfig))

	return ses.New(sess)
}
