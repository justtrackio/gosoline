package sqs

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cloudAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	MetadataKeyQueues = "cloud.aws.sqs.queues"
	sqsBatchSize      = 10
)

//go:generate mockery --name Queue
type Queue interface {
	GetName() string
	GetUrl() string
	GetArn() string

	DeleteMessage(ctx context.Context, receiptHandle string) error
	DeleteMessageBatch(ctx context.Context, receiptHandles []string) error
	Receive(ctx context.Context, maxNumberOfMessages int32, waitTime int32) ([]types.Message, error)
	Send(ctx context.Context, msg *Message) error
	SendBatch(ctx context.Context, messages []*Message) error
}

type QueueMetadata struct {
	AwsClientName string `json:"aws_client_name"`
	QueueArn      string `json:"queue_arn"`
	QueueName     string `json:"queue_name"`
	QueueNameFull string `json:"queue_name_full"`
	QueueUrl      string `json:"queue_url"`
}

type Message struct {
	DelaySeconds           int32
	MessageGroupId         *string
	MessageDeduplicationId *string
	Body                   *string
}

type FifoSettings struct {
	Enabled                   bool `cfg:"enabled" default:"false"`
	ContentBasedDeduplication bool `cfg:"content_based_deduplication" default:"false"`
}

type RedrivePolicy struct {
	Enabled         bool `cfg:"enabled" default:"true"`
	MaxReceiveCount int  `cfg:"max_receive_count" default:"3"`
}

type Properties struct {
	Name string
	Url  string
	Arn  string
}

type Settings struct {
	QueueName         string
	VisibilityTimeout int
	Fifo              FifoSettings
	RedrivePolicy     RedrivePolicy
	ClientName        string
}

type queue struct {
	logger     log.Logger
	client     Client
	uuidGen    uuid.Uuid
	properties *Properties
}

type metadataQueueKey string

func ProvideQueue(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, optFns ...ClientOption) (Queue, error) {
	key := fmt.Sprintf("%s-%s", settings.ClientName, settings.QueueName)

	return appctx.Provide(ctx, metadataQueueKey(key), func() (Queue, error) {
		return NewQueue(ctx, config, logger, settings, optFns...)
	})
}

func NewQueue(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, optFns ...ClientOption) (Queue, error) {
	var err error
	var client Client
	var props *Properties = &Properties{}

	if client, err = ProvideClient(ctx, config, logger, settings.ClientName, optFns...); err != nil {
		return nil, fmt.Errorf("can not create sqs client %s: %w", settings.ClientName, err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManager(settings, props, optFns...)); err != nil {
		return nil, fmt.Errorf("could not add lifecycle for sqs queue %s: %w", settings.QueueName, err)
	}

	return NewQueueWithInterfaces(logger, client, props), nil
}

func NewQueueWithInterfaces(logger log.Logger, client Client, props *Properties) Queue {
	return &queue{
		logger:     logger,
		client:     client,
		uuidGen:    uuid.New(),
		properties: props,
	}
}

func (q *queue) Send(ctx context.Context, msg *Message) error {
	input := &sqs.SendMessageInput{
		QueueUrl:               aws.String(q.properties.Url),
		DelaySeconds:           msg.DelaySeconds,
		MessageGroupId:         msg.MessageGroupId,
		MessageDeduplicationId: msg.MessageDeduplicationId,
		MessageBody:            msg.Body,
	}

	ctx = cloudAws.WithResourceTarget(ctx, q.properties.Name)

	_, err := q.client.SendMessage(ctx, input)

	return err
}

func (q *queue) SendBatch(ctx context.Context, messages []*Message) error {
	logger := q.logger.WithContext(ctx)
	if len(messages) == 0 {
		return nil
	}

	entries := make([]types.SendMessageBatchRequestEntry, len(messages))

	for i := 0; i < len(messages); i++ {
		id := q.uuidGen.NewV4()

		entries[i] = types.SendMessageBatchRequestEntry{
			Id:                     aws.String(id),
			DelaySeconds:           messages[i].DelaySeconds,
			MessageGroupId:         messages[i].MessageGroupId,
			MessageDeduplicationId: messages[i].MessageDeduplicationId,
			MessageBody:            messages[i].Body,
		}
	}

	input := &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(q.properties.Url),
		Entries:  entries,
	}

	ctx = cloudAws.WithResourceTarget(ctx, q.properties.Name)

	_, err := q.client.SendMessageBatch(ctx, input)

	var errRequestTooLong *types.BatchRequestTooLong
	if errors.As(err, &errRequestTooLong) && len(messages) > 1 {
		logger.Info("messages were bigger than the allowed max, splitting them up")

		half := float64(len(messages)) / 2
		chunkSize := int(math.Ceil(half))
		messageChunks := funk.Chunk(messages, chunkSize)

		for _, msgChunk := range messageChunks {
			if err := q.SendBatch(ctx, msgChunk); err != nil {
				return err
			}
		}

		return nil
	}

	return err
}

func (q *queue) Receive(ctx context.Context, maxNumberOfMessages int32, waitTime int32) ([]types.Message, error) {
	input := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []string{"ALL"},
		MaxNumberOfMessages:   maxNumberOfMessages,
		QueueUrl:              aws.String(q.properties.Url),
		WaitTimeSeconds:       waitTime,
	}

	ctx = cloudAws.WithResourceTarget(ctx, q.properties.Name)

	var err error
	var out *sqs.ReceiveMessageOutput

	if out, err = q.client.ReceiveMessage(ctx, input); err != nil {
		if exec.IsRequestCanceled(err) {
			return nil, nil
		}

		return nil, err
	}

	return out.Messages, nil
}

func (q *queue) DeleteMessage(ctx context.Context, receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.properties.Url),
		ReceiptHandle: aws.String(receiptHandle),
	}

	ctx = cloudAws.WithResourceTarget(ctx, q.properties.Name)

	_, err := q.client.DeleteMessage(ctx, input)

	return err
}

func (q *queue) DeleteMessageBatch(ctx context.Context, receiptHandles []string) error {
	input := &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(q.properties.Url),
	}

	entries := make([]types.DeleteMessageBatchRequestEntry, len(receiptHandles))

	for i, receiptHandle := range receiptHandles {
		entries[i] = types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(q.uuidGen.NewV4()),
			ReceiptHandle: aws.String(receiptHandle),
		}
	}

	ctx = cloudAws.WithResourceTarget(ctx, q.properties.Name)

	multiError := new(multierror.Error)

	for i := 0; i < len(entries); i += sqsBatchSize {
		j := i + sqsBatchSize

		if j > len(entries) {
			j = len(entries)
		}

		input.Entries = entries[i:j]

		if _, err := q.client.DeleteMessageBatch(ctx, input); err != nil {
			multiError = multierror.Append(multiError, err)
		}
	}

	return multiError.ErrorOrNil()
}

func (q *queue) GetName() string {
	return q.properties.Name
}

func (q *queue) GetUrl() string {
	return q.properties.Url
}

func (q *queue) GetArn() string {
	return q.properties.Arn
}
