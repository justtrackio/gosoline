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

func GetClient(config cfg.Config, logger mon.Logger) sqsiface.SQSAPI {
	c.Lock()
	defer c.Unlock()

	if c.initialized {
		return c.client
	}

	c.client = buildClient(config, logger)
	c.initialized = true

	return c.client
}

func buildClient(config cfg.Config, logger mon.Logger) *sqs.SQS {
	endpoint := config.GetString("aws_sqs_endpoint")
	maxRetries := config.GetInt("aws_sdk_retries")

	awsConfig := cloud.ConfigTemplate
	awsConfig.WithEndpoint(endpoint)
	awsConfig.WithMaxRetries(maxRetries)
	awsConfig.WithLogger(cloud.PrefixedLogger(logger, "sqs"))

	sess := session.Must(session.NewSession(awsConfig))

	return sqs.New(sess)
}
