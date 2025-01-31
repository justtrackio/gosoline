package sns

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cloudAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const (
	MaxBatchSize      = 10
	MetadataKeyTopics = "cloud.aws.sns.topics"
)

//go:generate mockery --name Topic
type Topic interface {
	Publish(ctx context.Context, msg string, attributes ...map[string]string) error
	PublishBatch(ctx context.Context, messages []string, attributes []map[string]string) error
}

type TopicMetadata struct {
	AwsClientName string `json:"aws_client_name"`
	TopicArn      string `json:"topic_arn"`
	TopicName     string `json:"topic_name"`
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

	if client, err = ProvideClient(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("can not create sns client %s: %w", settings.ClientName, err)
	}

	topic := NewTopicWithInterfaces(logger, client, "")
	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManager(settings, &topic.topicArn)); err != nil {
		return nil, fmt.Errorf("can not add lifecycle manager: %w", err)
	}

	return topic, nil
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
		TopicArn:          &t.topicArn,
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
		TopicArn:                   &t.topicArn,
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
