package sqs

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"sync"
)

var client = struct {
	sync.Mutex
	instance *sqs.SQS
}{}

func ProvideClient(config cfg.Config, logger log.Logger, settings *Settings) *sqs.SQS {
	client.Lock()
	defer client.Unlock()

	if client.instance != nil {
		return client.instance
	}

	client.instance = NewClient(config, logger, settings)

	return client.instance
}

func NewClient(config cfg.Config, logger log.Logger, settings *Settings) *sqs.SQS {
	if settings.Backoff.Enabled {
		settings.Client.MaxRetries = 0
	}

	awsConfig := cloud.GetAwsConfig(config, logger, "sqs", &settings.Client)
	sess := session.Must(session.NewSession(awsConfig))

	return sqs.New(sess)
}
