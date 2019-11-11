package sqs

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/hashicorp/go-multierror"
	"github.com/jonboulle/clockwork"
	"github.com/twinj/uuid"
	"net"
	"net/url"
	"os"
	"syscall"
)

const (
	sqsBatchSize              = 10
	MetricNameQueueErrorCount = "QueueErrorCount"
)

//go:generate mockery -name Queue
type Queue interface {
	GetName() string
	GetUrl() string
	GetArn() string

	DeleteMessage(receiptHandle string) error
	DeleteMessageBatch(receiptHandles []string) error
	Receive(ctx context.Context, waitTime int64) ([]*sqs.Message, error)
	Send(ctx context.Context, msg *Message) error
	SendBatch(ctx context.Context, messages []*Message) error
}

type Message struct {
	DelaySeconds   *int64
	MessageGroupId *string
	Body           *string
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
	Fifo              FifoSettings
	VisibilityTimeout int
	RedrivePolicy     RedrivePolicy
}

type queue struct {
	logger     mon.Logger
	client     sqsiface.SQSAPI
	properties *Properties
	metric     mon.MetricWriter
	clock      clockwork.Clock
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

	defaults := getDefaultQueueMetrics(props.Name)
	metric := mon.NewMetricDaemonWriter(defaults...)

	return NewWithInterfaces(logger, c, props, metric, clockwork.NewRealClock())
}

func NewWithInterfaces(logger mon.Logger, c sqsiface.SQSAPI, p *Properties, m mon.MetricWriter, cl clockwork.Clock) *queue {
	q := &queue{
		logger:     logger,
		client:     c,
		properties: p,
		metric:     m,
		clock:      cl,
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
		q.writeMetric(MetricNameQueueErrorCount, 1)
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
		q.writeMetric(MetricNameQueueErrorCount, 1)
		q.logger.WithContext(ctx).Errorf(err, "could not send batch to sqs queue %s", q.properties.Name)
	}

	return err
}

func (q *queue) Receive(ctx context.Context, waitTime int64) ([]*sqs.Message, error) {
	input := &sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{aws.String("ALL")},
		MaxNumberOfMessages:   aws.Int64(10),
		QueueUrl:              aws.String(q.properties.Url),
		WaitTimeSeconds:       aws.Int64(waitTime),
	}

	out, err := q.client.ReceiveMessageWithContext(ctx, input)

	if isError(err, request.CanceledErrorCode) {
		return nil, nil
	}

	if isConnResetError(err) {
		// write to cloud watch to keep track of these errors, but don't sound an alarm immediately
		q.writeMetric(MetricNameQueueErrorCount, 1)

		return nil, nil
	}

	if err != nil {
		q.writeMetric(MetricNameQueueErrorCount, 1)
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
		q.writeMetric(MetricNameQueueErrorCount, 1)
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

		_, err := q.client.DeleteMessageBatch(input)

		if err != nil {
			q.writeMetric(MetricNameQueueErrorCount, 1)
			q.logger.Errorf(err, "could not delete the messages from sqs queue %s", q.properties.Name)

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

func (q *queue) writeMetric(metric string, count int) {
	q.metric.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  q.clock.Now(),
		MetricName: metric,
		Dimensions: map[string]string{
			"Queue": q.GetName(),
		},
		Value: float64(count),
		Unit:  mon.UnitCount,
	})
}

func isError(err error, awsCode string) bool {
	if err == nil {
		return false
	}

	aerr, ok := err.(awserr.Error)

	return ok && aerr.Code() == awsCode
}

func isConnResetError(err error) bool {
	if err == nil {
		return false
	}

	aerr, ok := err.(awserr.Error)

	if !ok {
		return false
	}

	urlErr, ok := aerr.OrigErr().(*url.Error)

	if !ok {
		return false
	}

	opErr, ok := urlErr.Err.(*net.OpError)

	if !ok {
		return false
	}

	for {
		if nextOpErr, ok := opErr.Err.(*net.OpError); ok {
			opErr = nextOpErr
		} else {
			break
		}
	}

	syscallErr, ok := opErr.Err.(*os.SyscallError)

	if !ok {
		return false
	}

	return syscallErr.Err == syscall.ECONNRESET || syscallErr.Err == syscall.EPIPE
}

func getDefaultQueueMetrics(queueName string) mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: MetricNameQueueErrorCount,
			Dimensions: map[string]string{
				"Queue": queueName,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
