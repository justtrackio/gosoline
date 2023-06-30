package sns

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cloudAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

const MaxBatchSize = 10

//go:generate mockery --name Topic
type Topic interface {
	Publish(ctx context.Context, msg string, attributes ...map[string]string) error
	PublishBatch(ctx context.Context, messages []string, attributes []map[string]string) error
	SubscribeSqs(ctx context.Context, queueArn string, attributes map[string]string) error
}

type TopicSettings struct {
	TopicName  string
	ClientName string
}

type snsTopic struct {
	logger   log.Logger
	client   Client
	topicArn string
}

func NewTopic(ctx context.Context, config cfg.Config, logger log.Logger, settings *TopicSettings) (*snsTopic, error) {
	var err error
	var client Client
	var topicArn string

	if client, err = ProvideClient(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("can not create sns client %s: %w", settings.ClientName, err)
	}

	if topicArn, err = CreateTopic(ctx, logger, client, settings.TopicName); err != nil {
		return nil, fmt.Errorf("can not create sns topic %s: %w", settings.TopicName, err)
	}

	return NewTopicWithInterfaces(logger, client, topicArn), nil
}

func NewTopicWithInterfaces(logger log.Logger, client Client, topicArn string) *snsTopic {
	return &snsTopic{
		logger:   logger,
		client:   client,
		topicArn: topicArn,
	}
}

func (t *snsTopic) Publish(ctx context.Context, msg string, attributes ...map[string]string) error {
	inputAttributes, err := buildAttributes(attributes)
	if err != nil {
		return fmt.Errorf("can not build message attributes: %w", err)
	}

	input := &sns.PublishInput{
		TopicArn:          aws.String(t.topicArn),
		Message:           aws.String(msg),
		MessageAttributes: inputAttributes,
	}

	ctx = cloudAws.WithResourceTarget(ctx, t.topicArn)

	_, err = t.client.Publish(ctx, input)

	if exec.IsRequestCanceled(err) {
		t.logger.WithContext(ctx).WithFields(log.Fields{
			"arn": t.topicArn,
		}).Info("request was canceled while publishing to topic")

		return fmt.Errorf("request was canceled while publishing to topic: %w", err)
	}

	if err != nil {
		return fmt.Errorf("could not publish message: %w", err)
	}

	return nil
}

// PublishBatch fails at the first batch that could not be published.
func (t *snsTopic) PublishBatch(ctx context.Context, messages []string, attributes []map[string]string) error {
	if len(messages) != len(attributes) {
		return fmt.Errorf("there should be as many attributes as messages")
	}

	entries, err := t.computeEntries(messages, attributes)
	if err != nil {
		return fmt.Errorf("could not compute entries: %w", err)
	}

	ctx = cloudAws.WithResourceTarget(ctx, t.topicArn)

	for i := 0; i < len(messages); i += MaxBatchSize {
		currentBatch := t.getSubSlice(i, entries)

		err := t.publishSubSlice(ctx, currentBatch)
		if err != nil {
			return fmt.Errorf("could not publish sub slice [%d, %d]:%w", i, i+len(currentBatch)-1, err)
		}
	}

	return nil
}

func (t *snsTopic) getSubSlice(from int, entries []types.PublishBatchRequestEntry) []types.PublishBatchRequestEntry {
	endIndex := from + MaxBatchSize
	if endIndex >= len(entries) {
		endIndex = len(entries)
	}

	return entries[from:endIndex]
}

func (t *snsTopic) publishSubSlice(ctx context.Context, entries []types.PublishBatchRequestEntry) error {
	if len(entries) > MaxBatchSize {
		return fmt.Errorf("batch is too large, max length is %d", MaxBatchSize)
	}

	input := &sns.PublishBatchInput{
		TopicArn:                   aws.String(t.topicArn),
		PublishBatchRequestEntries: entries,
	}

	_, err := t.client.PublishBatch(ctx, input)

	if exec.IsRequestCanceled(err) {
		t.logger.WithContext(ctx).WithFields(log.Fields{
			"arn": t.topicArn,
		}).Info("request was canceled while publishing to topic")

		return fmt.Errorf("request was canceled while publishing to topic: %w", err)
	}

	if err != nil {
		return fmt.Errorf("could not publish batch: %w", err)
	}

	return nil
}

func (t *snsTopic) computeEntries(messages []string, attributes []map[string]string) ([]types.PublishBatchRequestEntry, error) {
	result := make([]types.PublishBatchRequestEntry, len(messages))

	for i := 0; i < len(messages); i++ {
		messageAttributes, err := buildAttributes([]map[string]string{attributes[i]})
		if err != nil {
			return nil, fmt.Errorf("could not build attributes for message %d: %w", i, err)
		}

		result[i] = types.PublishBatchRequestEntry{
			Id:                mdl.Box(strconv.Itoa(i)),
			Message:           &messages[i],
			MessageAttributes: messageAttributes,
		}
	}

	return result, nil
}

func (t *snsTopic) SubscribeSqs(ctx context.Context, queueArn string, attributes map[string]string) error {
	ctx = cloudAws.WithResourceTarget(ctx, t.topicArn)

	exists, err := t.subscriptionExists(ctx, queueArn, attributes)
	if err != nil {
		return fmt.Errorf("can not check if the subscription exists already: %w", err)
	}

	if exists {
		t.logger.WithFields(log.Fields{
			"topicArn": t.topicArn,
			"queueArn": queueArn,
		}).Info("already subscribed to sns topic")

		return nil
	}

	input := &sns.SubscribeInput{
		Attributes: map[string]string{},
		TopicArn:   aws.String(t.topicArn),
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

	_, err = t.client.Subscribe(ctx, input)

	if err != nil {
		return fmt.Errorf("could not subscribe to topic arn %s for sqs queue arn %s: %w", t.topicArn, queueArn, err)
	}

	t.logger.WithFields(log.Fields{
		"topicArn": t.topicArn,
		"queueArn": queueArn,
	}).Info("successful subscribed to sns topic")

	return nil
}

func (t *snsTopic) subscriptionExists(ctx context.Context, queueArn string, attributes map[string]string) (bool, error) {
	var ok bool
	var err error
	var subscriptions []types.Subscription

	if subscriptions, err = t.listSubscriptions(ctx); err != nil {
		return false, err
	}

	for _, subscription := range subscriptions {
		if *subscription.Endpoint != queueArn {
			continue
		}

		if ok, err = t.subscriptionAttributesMatch(ctx, subscription.SubscriptionArn, attributes); err != nil {
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

		if err = t.deleteSubscription(ctx, subscription.SubscriptionArn); err != nil {
			return false, fmt.Errorf("can not delete subscription: %w", err)
		}
	}

	return false, nil
}

func (t *snsTopic) listSubscriptions(ctx context.Context) ([]types.Subscription, error) {
	subscriptions := make([]types.Subscription, 0)

	input := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(t.topicArn),
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

func (t *snsTopic) subscriptionAttributesMatch(ctx context.Context, subscriptionArn *string, attributes map[string]string) (bool, error) {
	var ok bool
	var err error
	var subAttributes map[string]string
	var actualFilterPolicy string
	var expectedFilterPolicy []byte
	var actualAttributes, expectedAttributes map[string]interface{}

	if subAttributes, err = t.getSubscriptionAttributes(ctx, subscriptionArn); err != nil {
		return false, err
	}

	if actualFilterPolicy, ok = subAttributes["FilterPolicy"]; !ok {
		actualFilterPolicy = "null"
	}

	if err = json.Unmarshal([]byte(actualFilterPolicy), &actualAttributes); err != nil {
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

	matches := reflect.DeepEqual(expectedAttributes, actualAttributes)

	return matches, nil
}

func (t *snsTopic) getSubscriptionAttributes(ctx context.Context, subscriptionArn *string) (map[string]string, error) {
	input := &sns.GetSubscriptionAttributesInput{
		SubscriptionArn: subscriptionArn,
	}

	var err error
	var out *sns.GetSubscriptionAttributesOutput

	if out, err = t.client.GetSubscriptionAttributes(ctx, input); err != nil {
		return nil, fmt.Errorf("can not get subscription attributes: %w", err)
	}

	return out.Attributes, nil
}

func (t *snsTopic) deleteSubscription(ctx context.Context, subscriptionArn *string) error {
	input := &sns.UnsubscribeInput{
		SubscriptionArn: subscriptionArn,
	}

	_, err := t.client.Unsubscribe(ctx, input)

	return err
}
