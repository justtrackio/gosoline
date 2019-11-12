package sns

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

//go:generate mockery -name Topic
type Topic interface {
	Publish(ctx context.Context, msg *string) error
}

type Settings struct {
	cfg.AppId
	TopicId string
	Arn     string
	Client  cloud.ClientSettings
	Backoff cloud.BackoffSettings
}

type snsTopic struct {
	logger   mon.Logger
	client   snsiface.SNSAPI
	executor cloud.RequestExecutor
	settings *Settings
}

func NewTopic(config cfg.Config, logger mon.Logger, s *Settings) *snsTopic {
	s.PadFromConfig(config)

	client := GetClient(config, logger, &s.Client)

	arn, err := CreateTopic(logger, client, s)

	if err != nil {
		logger.Fatalf(err, "can not create sns topic %s", s.TopicId)
	}

	s.Arn = arn

	res := &cloud.BackoffResource{
		Type: "sns",
		Name: namingStrategy(s.AppId, s.TopicId),
	}
	executor := cloud.NewBackoffExecutor(logger, res, &s.Backoff)

	return NewTopicWithInterfaces(logger, client, executor, s)
}

func NewTopicWithInterfaces(logger mon.Logger, client snsiface.SNSAPI, executor cloud.RequestExecutor, s *Settings) *snsTopic {
	return &snsTopic{
		logger:   logger,
		client:   client,
		executor: executor,
		settings: s,
	}
}

func (t *snsTopic) Publish(ctx context.Context, msg *string) error {
	input := &sns.PublishInput{
		TopicArn: aws.String(t.settings.Arn),
		Message:  msg,
	}

	_, err := t.executor.Execute(ctx, func(delayedCtx context.Context) (interface{}, error) {
		return t.client.PublishWithContext(delayedCtx, input)
	})

	if err != nil {
		t.logger.WithFields(mon.Fields{
			"arn": t.settings.Arn,
		}).Error(err, "could not publish message to topic")
	}

	return err
}

func (t *snsTopic) SubscribeSqs(queueArn string) error {
	exists, err := t.subscriptionExists(queueArn)

	if err != nil {
		t.logger.WithFields(mon.Fields{
			"topicArn": t.settings.Arn,
			"queueArn": queueArn,
		}).Error(err, "can not check if subscription already exists")

		return err
	}

	if exists {
		t.logger.WithFields(mon.Fields{
			"topicArn": t.settings.Arn,
			"queueArn": queueArn,
		}).Info("already subscribed to sns topic")

		return nil
	}

	input := &sns.SubscribeInput{
		TopicArn: aws.String(t.settings.Arn),
		Endpoint: aws.String(queueArn),
		Protocol: aws.String("sqs"),
	}

	_, err = t.client.Subscribe(input)

	if err != nil {
		t.logger.WithFields(mon.Fields{
			"topicArn": t.settings.Arn,
			"queueArn": queueArn,
		}).Error(err, "could not subscribe for sqs queue")
	}

	t.logger.WithFields(mon.Fields{
		"topicArn": t.settings.Arn,
		"queueArn": queueArn,
	}).Info("successful subscribed to sns topic")

	return err
}

func (t *snsTopic) subscriptionExists(queueArn string) (bool, error) {
	subscriptions, err := t.listSubscriptions()

	if err != nil {
		return false, err
	}

	for _, s := range subscriptions {
		if *s.Endpoint == queueArn {
			return true, nil
		}
	}

	return false, nil
}

func (t *snsTopic) listSubscriptions() ([]*sns.Subscription, error) {
	subscriptions := make([]*sns.Subscription, 0)

	input := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(t.settings.Arn),
	}

	for {
		out, err := t.client.ListSubscriptionsByTopic(input)

		if err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, out.Subscriptions...)

		if out.NextToken == nil {
			break
		}

		input.NextToken = out.NextToken
	}

	return subscriptions, nil
}
