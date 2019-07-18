package sqs

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/twinj/uuid"
	"sync"
)

//go:generate mockery -name Queue
type Queue interface {
	GetName() string
	GetUrl() string
	GetArn() string

	DeleteMessage(msg *sqs.Message) error
	Receive(waitTime int64) ([]*sqs.Message, error)
	Send(ctx context.Context, value string) error
	SendBatch(ctx context.Context, values []string) error
}

type Settings struct {
	cfg.AppId
	QueueId string

	AutoCreate bool
	Url        string
	Arn        string
}

type queue struct {
	m sync.Mutex

	logger mon.Logger
	client sqsiface.SQSAPI

	name string
	url  string
	arn  string
}

func New(config cfg.Config, logger mon.Logger, s Settings) *queue {
	var err error

	name := namingStrategy(s.AppId, s.QueueId)
	c := GetClient(config, logger)

	s.PadFromConfig(config)
	s.AutoCreate = config.GetBool("aws_sqs_autoCreate")

	CreateQueue(logger, c, s)

	if s.Url, err = GetUrl(logger, c, s); err != nil {
		logger.Fatalf(err, "could not get url of queue %s", name)
	}

	if s.Arn, err = GetArn(logger, c, s); err != nil {
		logger.Fatalf(err, "could not get arn of queue %s", name)
	}

	return NewWithInterfaces(logger, c, s)
}

func NewWithInterfaces(logger mon.Logger, c sqsiface.SQSAPI, s Settings) *queue {
	name := namingStrategy(s.AppId, s.QueueId)

	q := &queue{
		logger: logger,
		client: c,
		name:   name,
		url:    s.Url,
		arn:    s.Arn,
	}

	return q
}

func (q *queue) Send(ctx context.Context, value string) error {
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(q.url),
		MessageBody: aws.String(value),
	}

	_, err := q.client.SendMessageWithContext(ctx, input)

	if err != nil {
		q.logger.WithContext(ctx).Error(err, "could not send value to sqs queue")
	}

	return err
}

func (q *queue) SendBatch(ctx context.Context, values []string) error {
	entries := make([]*sqs.SendMessageBatchRequestEntry, len(values))

	for i := 0; i < len(values); i++ {
		id := uuid.NewV4().String()

		entries[i] = &sqs.SendMessageBatchRequestEntry{
			Id:          aws.String(id),
			MessageBody: aws.String(values[i]),
		}
	}

	input := &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(q.url),
		Entries:  entries,
	}

	_, err := q.client.SendMessageBatchWithContext(ctx, input)

	if err != nil {
		q.logger.WithContext(ctx).Error(err, "could not send batch to sqs queue")
	}

	return err
}

func (q *queue) Receive(waitTime int64) ([]*sqs.Message, error) {
	input := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{aws.String("ALL")},
		MaxNumberOfMessages:   aws.Int64(1),
		QueueUrl:              aws.String(q.url),
		WaitTimeSeconds:       aws.Int64(waitTime),
	}

	out, err := q.client.ReceiveMessage(input)

	if err != nil {
		q.logger.Error(err, "could not receive value from sqs queue")
		return nil, err
	}

	return out.Messages, nil
}

func (q *queue) DeleteMessage(msg *sqs.Message) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.url),
		ReceiptHandle: msg.ReceiptHandle,
	}

	_, err := q.client.DeleteMessage(input)

	if err != nil {
		q.logger.Error(err, "could not delete message from sqs queue")
		return err
	}

	return nil
}

func (q *queue) GetName() string {
	return q.name
}

func (q *queue) GetUrl() string {
	return q.url
}

func (q *queue) GetArn() string {
	return q.arn
}
