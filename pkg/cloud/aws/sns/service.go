package sns

import (
	"context"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cloudAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Service struct {
	logger log.Logger
	client Client
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger, clientName string) (*Service, error) {
	var err error
	var client Client

	if client, err = ProvideClient(ctx, config, logger, clientName); err != nil {
		return nil, fmt.Errorf("can not create sns client %s: %w", clientName, err)
	}

	return NewServiceWithInterfaces(logger, client), nil
}

func NewServiceWithInterfaces(logger log.Logger, client Client) *Service {
	return &Service{logger: logger, client: client}
}

func (s *Service) CreateTopic(ctx context.Context, topicName string) (string, error) {
	s.logger.WithFields(log.Fields{
		"name": topicName,
	}).Info(ctx, "looking for sns topic")

	input := &sns.CreateTopicInput{
		Name: aws.String(topicName),
	}

	var err error
	var out *sns.CreateTopicOutput

	if out, err = s.client.CreateTopic(ctx, input); err != nil {
		return "", err
	}

	s.logger.WithFields(log.Fields{
		"name": topicName,
		"arn":  *out.TopicArn,
	}).Info(ctx, "found sns topic")

	return *out.TopicArn, nil
}

func (s *Service) SubscribeSqs(ctx context.Context, queueArn string, topicArn string, attributes map[string]string) error {
	ctx = cloudAws.WithResourceTarget(ctx, topicArn)

	var err error
	var exists bool

	if exists, err = s.subscriptionExists(ctx, queueArn, topicArn, attributes); err != nil {
		return fmt.Errorf("can not check if the subscription exists already: %w", err)
	}

	if exists {
		s.logger.WithFields(log.Fields{
			"topicArn": topicArn,
			"queueArn": queueArn,
		}).Info(ctx, "already subscribed to sns topic")

		return nil
	}

	input := &sns.SubscribeInput{
		Attributes: map[string]string{},
		TopicArn:   aws.String(topicArn),
		Endpoint:   aws.String(queueArn),
		Protocol:   aws.String("sqs"),
	}

	if len(attributes) > 0 {
		policy, err := buildFilterPolicy(attributes)
		if err != nil {
			return fmt.Errorf("can not build filter policy: %w", err)
		}

		input.Attributes["FilterPolicy"] = policy
	}

	if _, err = s.client.Subscribe(ctx, input); err != nil {
		return fmt.Errorf("could not subscribe to topic arn %s for sqs queue arn %s: %w", topicArn, queueArn, err)
	}

	s.logger.WithFields(log.Fields{
		"topicArn": topicArn,
		"queueArn": queueArn,
	}).Info(ctx, "successful subscribed to sns topic")

	return nil
}

func (s *Service) subscriptionExists(ctx context.Context, queueArn string, topicArn string, attributes map[string]string) (bool, error) {
	var ok bool
	var err error
	var subscriptions []types.Subscription

	if subscriptions, err = s.listSubscriptions(ctx, topicArn); err != nil {
		return false, err
	}

	for _, subscription := range subscriptions {
		if *subscription.Endpoint != queueArn {
			continue
		}

		if ok, err = s.subscriptionAttributesMatch(ctx, subscription.SubscriptionArn, attributes); err != nil {
			return false, err
		}

		if ok {
			return true, nil
		}

		s.logger.WithFields(log.Fields{
			"topicArn":        *subscription.TopicArn,
			"subscriptionArt": *subscription.SubscriptionArn,
			"queueArn":        queueArn,
		}).Info(ctx, "found not matching subscription for queue %s, deleting %s", queueArn, *subscription.SubscriptionArn)

		if err = s.deleteSubscription(ctx, subscription.SubscriptionArn); err != nil {
			return false, fmt.Errorf("can not delete subscription: %w", err)
		}
	}

	return false, nil
}

func (t *Service) listSubscriptions(ctx context.Context, topicArn string) ([]types.Subscription, error) {
	subscriptions := make([]types.Subscription, 0)

	input := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(topicArn),
	}

	var err error
	var out *sns.ListSubscriptionsByTopicOutput

	for {
		if out, err = t.client.ListSubscriptionsByTopic(ctx, input); err != nil {
			return nil, fmt.Errorf("can not list subscriptions by topic: %w", err)
		}

		subscriptions = append(subscriptions, out.Subscriptions...)

		if out.NextToken == nil {
			break
		}

		input.NextToken = out.NextToken
	}

	return subscriptions, nil
}

func (s *Service) subscriptionAttributesMatch(ctx context.Context, subscriptionArn *string, attributes map[string]string) (bool, error) {
	var ok bool
	var err error
	var subAttributes map[string]string
	var actualFilterPolicy string
	var expectedFilterPolicy []byte
	var actualAttributes, expectedAttributes map[string]any

	if subAttributes, err = s.getSubscriptionAttributes(ctx, subscriptionArn); err != nil {
		return false, err
	}

	if actualFilterPolicy, ok = subAttributes["FilterPolicy"]; !ok {
		actualFilterPolicy = "null"
	}

	if err = json.Unmarshal([]byte(actualFilterPolicy), &actualAttributes); err != nil {
		return false, fmt.Errorf("can not unmarshal actual filter policy: %w", err)
	}

	// we have to marshal and unmarshal this to cover the behavior of getting float64 for all numbers,
	// if we unmarshal something into a map[string]any
	if expectedFilterPolicy, err = json.Marshal(attributes); err != nil {
		return false, fmt.Errorf("can not marshal expected filter policy: %w", err)
	}

	if err = json.Unmarshal(expectedFilterPolicy, &expectedAttributes); err != nil {
		return false, fmt.Errorf("can not unmarshal expected filter policy: %w", err)
	}

	matches := reflect.DeepEqual(expectedAttributes, actualAttributes)

	return matches, nil
}

func (s *Service) getSubscriptionAttributes(ctx context.Context, subscriptionArn *string) (map[string]string, error) {
	input := &sns.GetSubscriptionAttributesInput{
		SubscriptionArn: subscriptionArn,
	}

	var err error
	var out *sns.GetSubscriptionAttributesOutput

	if out, err = s.client.GetSubscriptionAttributes(ctx, input); err != nil {
		return nil, fmt.Errorf("can not get subscription attributes: %w", err)
	}

	return out.Attributes, nil
}

func (s *Service) deleteSubscription(ctx context.Context, subscriptionArn *string) error {
	input := &sns.UnsubscribeInput{
		SubscriptionArn: subscriptionArn,
	}

	if _, err := s.client.Unsubscribe(ctx, input); err != nil {
		return fmt.Errorf("can not unsubscribe from sns topic: %w", err)
	}

	return nil
}
