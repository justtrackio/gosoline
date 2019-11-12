package sqs

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"sync"
)

type client struct {
	sync.Mutex
	client      sqsiface.SQSAPI
	initialized bool
}

var c = client{}

func GetClient(config cfg.Config, logger mon.Logger, settings *cloud.ClientSettings) sqsiface.SQSAPI {
	c.Lock()
	defer c.Unlock()

	if c.initialized {
		return c.client
	}

	c.client = buildClient(config, logger, settings)
	c.initialized = true

	return c.client
}

func buildClient(config cfg.Config, logger mon.Logger, settings *cloud.ClientSettings) *sqs.SQS {
	awsConfig := cloud.GetAwsConfig(config, logger, "sqs", settings)
	sess := session.Must(session.NewSession(awsConfig))

	return sqs.New(sess)
}
