package sqs

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/twinj/uuid"
)

//go:generate mockery -name Queue
type Queue interface {
	GetName() string
	GetUrl() string
	GetArn() string

	DeleteMessage(receiptHandle string) error
	Receive(waitTime int64) ([]*sqs.Message, error)
	Send(ctx context.Context, msg *Message) error
	SendBatch(ctx context.Context, messages []*Message) error
}

type Message struct {
	DelaySeconds   *int64
	MessageGroupId *string
	Body           *string
}

type FifoSettings struct {
	Enabled                   bool `mapstructure:"enabled"`
	ContentBasedDeduplication bool `mapstructure:"contentBasedDeduplication"`
}

type RedrivePolicy struct {
	Enabled         bool   `mapstructure:"enabled"`
	MaxReceiveCount int    `mapstructure:"max_receive_count"`
	QueueName       string `mapstructure:"queue_name"`
}

type Properties struct {
	Name string
	Url  string
	Arn  string
}

type Settings struct {
	cfg.AppId
	QueueId           string
	Fifo              FifoSettings
	VisibilityTimeout int
	RedrivePolicy     RedrivePolicy
}

type queue struct {
	logger     mon.Logger
	client     sqsiface.SQSAPI
	properties *Properties
}

func New(config cfg.Config, logger mon.Logger, s Settings) *queue {
	s.PadFromConfig(config)
	name := generateName(s)

	c := GetClient(config, logger)
	srv := NewService(config, logger)

	props, err := srv.CreateQueue(s)

	if err != nil {
		logger.Fatalf(err, "could not create or get properties of queue %s", name)
	}

	return NewWithInterfaces(logger, c, props)
}

func NewWithInterfaces(logger mon.Logger, c sqsiface.SQSAPI, p *Properties) *queue {
	q := &queue{
		logger:     logger,
		client:     c,
		properties: p,
	}

	return q
}

func (q *queue) Send(ctx context.Context, msg *Message) error {
	input := &sqs.SendMessageInput{
		QueueUrl:       aws.String(q.properties.Url),
		DelaySeconds:   msg.DelaySeconds,
		MessageGroupId: msg.MessageGroupId,
		MessageBody:    msg.Body,
	}

	_, err := q.client.SendMessageWithContext(ctx, input)

	if err != nil {
		q.logger.WithContext(ctx).Errorf(err, "could not send value to sqs queue %s", q.properties.Name)
	}

	return err
}

func (q *queue) SendBatch(ctx context.Context, messages []*Message) error {
	if len(messages) == 0 {
		return nil
	}

	entries := make([]*sqs.SendMessageBatchRequestEntry, len(messages))

	for i := 0; i < len(messages); i++ {
		id := uuid.NewV4().String()

		entries[i] = &sqs.SendMessageBatchRequestEntry{
			Id:             aws.String(id),
			DelaySeconds:   messages[i].DelaySeconds,
			MessageGroupId: messages[i].MessageGroupId,
			MessageBody:    messages[i].Body,
		}
	}

	input := &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(q.properties.Url),
		Entries:  entries,
	}

	_, err := q.client.SendMessageBatchWithContext(ctx, input)

	if err != nil {
		q.logger.WithContext(ctx).Errorf(err, "could not send batch to sqs queue %s", q.properties.Name)
	}

	return err
}

func (q *queue) Receive(waitTime int64) ([]*sqs.Message, error) {
	input := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{aws.String("ALL")},
		MaxNumberOfMessages:   aws.Int64(1),
		QueueUrl:              aws.String(q.properties.Url),
		WaitTimeSeconds:       aws.Int64(waitTime),
	}

	out, err := q.client.ReceiveMessage(input)

	if err != nil {
		q.logger.Errorf(err, "could not receive value from sqs queue %s", q.properties.Name)
		return nil, err
	}

	return out.Messages, nil
}

func (q *queue) DeleteMessage(receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.properties.Url),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := q.client.DeleteMessage(input)

	if err != nil {
		q.logger.Errorf(err, "could not delete message from sqs queue %s", q.properties.Name)
		return err
	}

	return nil
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
