package sqs

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/hashicorp/go-multierror"
	"github.com/thoas/go-funk"
	"github.com/twinj/uuid"
	"math"
)

const (
	sqsBatchSize      = 10
	sqsMaxMessageSize = 256 * 1024
)

//go:generate mockery -name Queue
type Queue interface {
	GetName() string
	GetUrl() string
	GetArn() string

	DeleteMessage(receiptHandle string) error
	DeleteMessageBatch(receiptHandles []string) error
	Receive(ctx context.Context, maxNumberOfMessages int64, waitTime int64) ([]*sqs.Message, error)
	Send(ctx context.Context, msg *Message) error
	SendBatch(ctx context.Context, messages []*Message) error
}

type Message struct {
	DelaySeconds           *int64
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
	cfg.AppId
	QueueId           string
	VisibilityTimeout int
	Fifo              FifoSettings
	RedrivePolicy     RedrivePolicy
	Client            cloud.ClientSettings
	Backoff           exec.BackoffSettings
}

type queue struct {
	logger     mon.Logger
	client     sqsiface.SQSAPI
	executor   gosoAws.Executor
	properties *Properties
}

func New(config cfg.Config, logger mon.Logger, settings *Settings) *queue {
	settings.PadFromConfig(config)
	name := generateName(settings)

	client := ProvideClient(config, logger, settings)
	srv := NewService(config, logger)

	props, err := srv.CreateQueue(settings)

	if err != nil {
		logger.Fatalf(err, "could not create or get properties of queue %s", name)
	}

	res := &exec.ExecutableResource{
		Type: "sqs",
		Name: name,
	}
	executor := gosoAws.NewExecutor(logger, res, &settings.Backoff)

	return NewWithInterfaces(logger, client, executor, props)
}

func NewWithInterfaces(logger mon.Logger, client sqsiface.SQSAPI, executor gosoAws.Executor, p *Properties) *queue {
	q := &queue{
		logger:     logger,
		client:     client,
		executor:   executor,
		properties: p,
	}

	return q
}

func (q *queue) Send(ctx context.Context, msg *Message) error {
	input := &sqs.SendMessageInput{
		QueueUrl:               aws.String(q.properties.Url),
		DelaySeconds:           msg.DelaySeconds,
		MessageGroupId:         msg.MessageGroupId,
		MessageDeduplicationId: msg.MessageDeduplicationId,
		MessageBody:            msg.Body,
	}

	_, err := q.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return q.client.SendMessageRequest(input)
	})

	if err != nil {
		q.logger.WithContext(ctx).Errorf(err, "could not send value to sqs queue %s", q.properties.Name)
	}

	return err
}

func (q *queue) SendBatch(ctx context.Context, messages []*Message) error {
	logger := q.logger.WithContext(ctx)
	if len(messages) == 0 {
		return nil
	}

	entries := make([]*sqs.SendMessageBatchRequestEntry, len(messages))

	for i := 0; i < len(messages); i++ {
		id := uuid.NewV4().String()

		entries[i] = &sqs.SendMessageBatchRequestEntry{
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

	_, err := q.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return q.client.SendMessageBatchRequest(input)
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == sqs.ErrCodeBatchRequestTooLong {
			logger.Info("messages were bigger than the allowed max, splitting them up")

			half := float64(len(messages)) / 2
			chunkSize := int(math.Ceil(half))
			msgs := funk.Chunk(messages, chunkSize).([][]*Message)

			for _, msgChunk := range msgs {
				err := q.SendBatch(ctx, msgChunk)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}

	if err != nil && !exec.IsRequestCanceled(err) {
		logger.Errorf(err, "could not send batch to sqs queue %s", q.properties.Name)
	}

	return err
}

func (q *queue) Receive(ctx context.Context, maxNumberOfMessages int64, waitTime int64) ([]*sqs.Message, error) {
	logger := q.logger.WithContext(ctx)

	input := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{aws.String("ALL")},
		MaxNumberOfMessages:   aws.Int64(maxNumberOfMessages),
		QueueUrl:              aws.String(q.properties.Url),
		WaitTimeSeconds:       aws.Int64(waitTime),
	}

	res, err := q.executor.Execute(ctx, func() (*request.Request, interface{}) {
		return q.client.ReceiveMessageRequest(input)
	})

	if exec.IsRequestCanceled(err) {
		logger.Warnf("canceled receive from sqs queue %s: %s", q.properties.Name, err.Error())
		return nil, nil
	}

	if err != nil {
		logger.Errorf(err, "could not receive value from sqs queue %s", q.properties.Name)
		return nil, err
	}

	out := res.(*sqs.ReceiveMessageOutput)

	return out.Messages, nil
}

func (q *queue) DeleteMessage(receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.properties.Url),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := q.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return q.client.DeleteMessageRequest(input)
	})

	if err != nil {
		q.logger.Errorf(err, "could not delete message from sqs queue %s", q.properties.Name)
		return err
	}

	return nil
}

func (q *queue) DeleteMessageBatch(receiptHandles []string) error {
	input := &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(q.properties.Url),
	}

	entries := make([]*sqs.DeleteMessageBatchRequestEntry, len(receiptHandles))

	for i, receiptHandle := range receiptHandles {
		entry := &sqs.DeleteMessageBatchRequestEntry{
			Id:            mdl.String(uuid.NewV4().String()),
			ReceiptHandle: mdl.String(receiptHandle),
		}

		entries[i] = entry
	}

	multiError := new(multierror.Error)

	for i := 0; i < len(entries); i += sqsBatchSize {
		j := i + sqsBatchSize

		if j > len(entries) {
			j = len(entries)
		}

		input.Entries = entries[i:j]

		err := q.doDeleteMessageBatch(input)

		if err != nil {
			multiError = multierror.Append(multiError, err)
		}
	}

	return multiError.ErrorOrNil()
}

func (q *queue) doDeleteMessageBatch(input *sqs.DeleteMessageBatchInput) error {
	_, err := q.executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return q.client.DeleteMessageBatchRequest(input)
	})

	if err != nil {
		q.logger.Errorf(err, "could not delete the messages from sqs queue %s", q.properties.Name)
	}

	return err
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
