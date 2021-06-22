package sns

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/thoas/go-funk"
)

//go:generate mockery -name Topic
type Topic interface {
	Publish(ctx context.Context, msg *string, attributes ...map[string]interface{}) error
	SubscribeSqs(queueArn string, attributes map[string]interface{}) error
}

type Settings struct {
	cfg.AppId
	TopicId string
	Arn     string
	Client  cloud.ClientSettings
	Backoff exec.BackoffSettings
}

type snsTopic struct {
	logger   log.Logger
	client   snsiface.SNSAPI
	executor gosoAws.Executor
	settings *Settings
}

func NewTopic(config cfg.Config, logger log.Logger, settings *Settings) (*snsTopic, error) {
	settings.PadFromConfig(config)

	client := ProvideClient(config, logger, settings)
	arn, err := CreateTopic(logger, client, settings)

	if err != nil {
		return nil, fmt.Errorf("can not create sns topic %s: %w", settings.TopicId, err)
	}

	settings.Arn = arn

	res := &exec.ExecutableResource{
		Type: "sns",
		Name: namingStrategy(settings.AppId, settings.TopicId),
	}
	executor := gosoAws.NewExecutor(logger, res, &settings.Backoff)

	return NewTopicWithInterfaces(logger, client, executor, settings), nil
}

func NewTopicWithInterfaces(logger log.Logger, client snsiface.SNSAPI, executor gosoAws.Executor, s *Settings) *snsTopic {
	return &snsTopic{
		logger:   logger,
		client:   client,
		executor: executor,
		settings: s,
	}
}

func (t *snsTopic) Publish(ctx context.Context, msg *string, attributes ...map[string]interface{}) error {
	inputAttributes, err := buildAttributes(attributes)

	if err != nil {
		return fmt.Errorf("can not build message attributes: %w", err)
	}

	input := &sns.PublishInput{
		TopicArn:          aws.String(t.settings.Arn),
		Message:           msg,
		MessageAttributes: inputAttributes,
	}

	_, err = t.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return t.client.PublishRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		t.logger.WithFields(log.Fields{
			"arn": t.settings.Arn,
		}).Info("request was canceled while publishing to topic")

		return err
	}

	return err
}

func (t *snsTopic) SubscribeSqs(queueArn string, attributes map[string]interface{}) error {
	exists, err := t.subscriptionExists(queueArn, attributes)

	if err != nil {
		return fmt.Errorf("can not check if the subscription exists already: %w", err)
	}

	if exists {
		t.logger.WithFields(log.Fields{
			"topicArn": t.settings.Arn,
			"queueArn": queueArn,
		}).Info("already subscribed to sns topic")

		return nil
	}

	input := &sns.SubscribeInput{
		Attributes: map[string]*string{},
		TopicArn:   aws.String(t.settings.Arn),
		Endpoint:   aws.String(queueArn),
		Protocol:   aws.String("sqs"),
	}

	if len(attributes) > 0 {
		policy, err := buildFilterPolicy(attributes)

		if err != nil {
			return fmt.Errorf("can not build filter policy: %w", err)
		}

		input.Attributes["FilterPolicy"] = aws.String(policy)
	}

	_, err = t.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return t.client.SubscribeRequest(input)
	})

	if err != nil {
		t.logger.WithFields(log.Fields{
			"topicArn": t.settings.Arn,
			"queueArn": queueArn,
		}).Error("could not subscribe for sqs queue: %w", err)
	}

	t.logger.WithFields(log.Fields{
		"topicArn": t.settings.Arn,
		"queueArn": queueArn,
	}).Info("successful subscribed to sns topic")

	return err
}

func (t *snsTopic) subscriptionExists(queueArn string, attributes map[string]interface{}) (bool, error) {
	var ok bool
	var err error
	var subscriptions []*sns.Subscription

	if subscriptions, err = t.listSubscriptions(); err != nil {
		return false, err
	}

	for _, subscription := range subscriptions {
		if *subscription.Endpoint != queueArn {
			continue
		}

		if ok, err = t.subscriptionAttributesMatch(subscription.SubscriptionArn, attributes); err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}

		t.logger.WithFields(log.Fields{
			"topicArn":        *subscription.TopicArn,
			"subscriptionArt": *subscription.SubscriptionArn,
			"queueArn":        queueArn,
		}).Info("found not matching subscription for queue %s, deleting %s", queueArn, *subscription.SubscriptionArn)

		if err = t.deleteSubscription(subscription.SubscriptionArn); err != nil {
			return false, fmt.Errorf("can not delete subscription: %w", err)
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

func (t *snsTopic) subscriptionAttributesMatch(subscriptionArn *string, attributes map[string]interface{}) (bool, error) {
	var ok bool
	var err error
	var subAttributes map[string]*string
	var actualFilterPolicy *string
	var expectedFilterPolicy []byte
	var actualAttributes, expectedAttributes map[string]interface{}

	if subAttributes, err = t.getSubscriptionAttributes(subscriptionArn); err != nil {
		return false, err
	}

	if actualFilterPolicy, ok = subAttributes["FilterPolicy"]; !ok {
		actualFilterPolicy = aws.String("null")
	}

	if err = json.Unmarshal([]byte(*actualFilterPolicy), &actualAttributes); err != nil {
		return false, fmt.Errorf("can not unmarshal actual filter policy: %w", err)
	}

	// we have to marshal and unmarshal this to cover the behavior of getting float64 for all numbers,
	// if we unmarshal something into a map[string]interface{}
	if expectedFilterPolicy, err = json.Marshal(attributes); err != nil {
		return false, fmt.Errorf("can not marshal expected filter policy: %w", err)
	}

	if err = json.Unmarshal(expectedFilterPolicy, &expectedAttributes); err != nil {
		return false, fmt.Errorf("can not unmarshal expected filter policy: %w", err)
	}

	matches := funk.IsEqual(expectedAttributes, actualAttributes)

	return matches, nil
}

func (t *snsTopic) getSubscriptionAttributes(subscriptionArn *string) (map[string]*string, error) {
	input := &sns.GetSubscriptionAttributesInput{
		SubscriptionArn: subscriptionArn,
	}

	outI, err := t.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return t.client.GetSubscriptionAttributesRequest(input)
	})

	if err != nil {
		return nil, fmt.Errorf("can not get subscription attributes: %w", err)
	}

	out := outI.(*sns.GetSubscriptionAttributesOutput)

	return out.Attributes, nil
}

func (t *snsTopic) deleteSubscription(subscriptionArn *string) error {
	input := &sns.UnsubscribeInput{
		SubscriptionArn: subscriptionArn,
	}

	_, err := t.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return t.client.UnsubscribeRequest(input)
	})

	return err
}
