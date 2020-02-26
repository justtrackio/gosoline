package sns

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
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

func NewTopic(config cfg.Config, logger mon.Logger, settings *Settings) *snsTopic {
	settings.PadFromConfig(config)

	client := ProvideClient(config, logger, settings)

	arn, err := CreateTopic(logger, client, settings)

	if err != nil {
		logger.Fatalf(err, "can not create sns topic %s", settings.TopicId)
	}

	settings.Arn = arn

	res := &cloud.BackoffResource{
		Type: "sns",
		Name: namingStrategy(settings.AppId, settings.TopicId),
	}
	executor := cloud.NewExecutor(logger, res, &settings.Backoff)

	return NewTopicWithInterfaces(logger, client, executor, settings)
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

	_, err := t.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return t.client.PublishRequest(input)
	})

	if cloud.IsRequestCanceled(err) {
		t.logger.WithFields(mon.Fields{
			"arn": t.settings.Arn,
		}).Info("request was canceled while publishing to topic")

		return err
	}

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

	_, err = t.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return t.client.SubscribeRequest(input)
	})

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
		outI, err := t.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
			return t.client.ListSubscriptionsByTopicRequest(input)
		})

		if err != nil {
			return nil, err
		}

		out := outI.(*sns.ListSubscriptionsByTopicOutput)

		subscriptions = append(subscriptions, out.Subscriptions...)

		if out.NextToken == nil {
			break
		}

		input.NextToken = out.NextToken
	}

	return subscriptions, nil
}
