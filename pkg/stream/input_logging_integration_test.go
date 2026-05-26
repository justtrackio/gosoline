package stream

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loggerOutput struct {
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields"`
}

func TestConfigurableInputLoggerFields_HasInputForEveryType(t *testing.T) {
	typesToTest := []string{
		InputTypeFile,
		InputTypeInMemory,
		InputTypeKafka,
		InputTypeKinesis,
		InputTypeRedis,
		InputTypeSns,
		InputTypeSqs,
	}

	for _, typ := range typesToTest {
		logger, buffer := newBufferedJsonLogger()
		config := cfg.New(map[string]any{
			"stream": map[string]any{
				"input": map[string]any{
					"test-input": map[string]any{
						"type": typ,
					},
				},
			},
		})

		oldFactory := inputFactories[typ]
		SetInputFactory(typ, func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Input, error) {
			logger.Info(ctx, "factory called")
			return NewNoopInput(), nil
		})
		defer SetInputFactory(typ, oldFactory)

		_, err := NewConfigurableInput(t.Context(), config, logger, "test-input")
		require.NoError(t, err)

		entry := lastLogOutput(t, buffer)
		assert.Equal(t, "test-input", entry.Fields["input"])
	}
}

func TestFileInputLoggerFields(t *testing.T) {
	logger, buffer := newBufferedJsonLogger()

	input := NewFileInputWithInterfaces(
		logger.WithFields(log.Fields{"input": "file-input"}),
		FileSettings{Filename: "payload.json"},
	).(*fileInput)

	input.logger.Info(t.Context(), "file logger test")

	entry := lastLogOutput(t, buffer)
	assert.Equal(t, "file-input", entry.Fields["input"])
	assert.Equal(t, "payload.json", entry.Fields["file_name"])
}

func TestRedisInputLoggerFields(t *testing.T) {
	logger, buffer := newBufferedJsonLogger()

	input := NewRedisListInputWithInterfaces(
		cfg.New(map[string]any{}),
		logger.WithFields(log.Fields{"input": "redis-input"}),
		nil,
		nil,
		&RedisListInputSettings{
			ServerName: "cache-main",
			Key:        "events",
		},
		nil,
	).(*redisListInput)

	input.logger.Info(t.Context(), "redis logger test")

	entry := lastLogOutput(t, buffer)
	assert.Equal(t, "redis-input", entry.Fields["input"])
	assert.Equal(t, "cache-main", entry.Fields["redis_server_name"])
	assert.Equal(t, "events", entry.Fields["redis_key"])
}

func TestKafkaInputLoggerFields(t *testing.T) {
	logger, buffer := newBufferedJsonLogger()

	config := cfg.New(map[string]any{
		"app": map[string]any{
			"project":   "proj",
			"family":    "fam",
			"group":     "grp",
			"name":      "svc",
			"env":       "test",
			"namespace": "proj-fam-grp-test",
		},
	})

	enrichedLogger, err := addKafkaInputLoggerFields(config, logger.WithFields(log.Fields{"input": "kafka-input"}), consumer.Settings{
		TopicId: "orders",
	})
	require.NoError(t, err)

	enrichedLogger.Info(t.Context(), "kafka logger test")

	entry := lastLogOutput(t, buffer)
	assert.Equal(t, "kafka-input", entry.Fields["input"])
	assert.NotEmpty(t, entry.Fields["kafka_topic"])
}

func TestSqsInputLoggerFields_AreFilledAfterQueuePropertiesLoad(t *testing.T) {
	logger, buffer := newBufferedJsonLogger()

	healthCheckTimer, err := clock.NewHealthCheckTimer(time.Second)
	require.NoError(t, err)

	queue := &queueTestDouble{}

	input := NewSqsInputWithInterfaces(
		logger.WithFields(log.Fields{"input": "sqs-input"}),
		queue,
		func(_ *string) (*Message, error) {
			return &Message{}, nil
		},
		healthCheckTimer,
		&SqsInputSettings{
			RunnerCount:         1,
			MaxNumberOfMessages: 1,
		},
	)

	queue.setProperties(
		"test-queue",
		"https://sqs.eu-west-1.amazonaws.com/123456789/test-queue",
		"arn:aws:sqs:eu-west-1:123456789:test-queue",
	)

	done := make(chan error, 1)
	go func() {
		done <- input.Run(t.Context())
	}()

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for sqs input message")
	case <-input.Data():
	}

	input.Stop(t.Context())

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for sqs input to stop")
	case err := <-done:
		require.NoError(t, err)
	}

	entry := logOutputByMessage(t, buffer, "starting sqs input")
	assert.Equal(t, "sqs-input", entry.Fields["input"])
	assert.Equal(t, "test-queue", entry.Fields["queue_name"])
	assert.Equal(t, "https://sqs.eu-west-1.amazonaws.com/123456789/test-queue", entry.Fields["queue_url"])
	assert.Equal(t, "arn:aws:sqs:eu-west-1:123456789:test-queue", entry.Fields["queue_arn"])
}

func newBufferedJsonLogger() (log.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	handler := log.NewHandlerIoWriter(cfg.New(map[string]any{}), log.PriorityTrace, log.FormatterJson, "main", time.RFC3339, buffer)

	return log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler}), buffer
}

func lastLogOutput(t *testing.T, buffer *bytes.Buffer) loggerOutput {
	t.Helper()

	lines := logLines(buffer)
	require.NotEmpty(t, lines)

	var out loggerOutput
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &out))

	return out
}

func logOutputByMessage(t *testing.T, buffer *bytes.Buffer, match string) loggerOutput {
	t.Helper()

	for _, line := range logLines(buffer) {
		out := loggerOutput{}
		require.NoError(t, json.Unmarshal([]byte(line), &out))

		if strings.Contains(out.Message, match) {
			return out
		}
	}

	t.Fatalf("expected log line containing %q", match)
	return loggerOutput{}
}

func logLines(buffer *bytes.Buffer) []string {
	all := strings.Split(buffer.String(), "\n")
	result := make([]string, 0, len(all))

	for _, line := range all {
		if line == "" {
			continue
		}

		result = append(result, line)
	}

	return result
}

type queueTestDouble struct {
	lock         sync.RWMutex
	name         string
	url          string
	arn          string
	receiveCount int32
}

func (q *queueTestDouble) setProperties(name, url, arn string) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.name = name
	q.url = url
	q.arn = arn
}

func (q *queueTestDouble) GetName() string {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return q.name
}

func (q *queueTestDouble) GetUrl() string {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return q.url
}

func (q *queueTestDouble) GetArn() string {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return q.arn
}

func (q *queueTestDouble) DeleteMessage(context.Context, string) error {
	return nil
}

func (q *queueTestDouble) DeleteMessageBatch(context.Context, []string) error {
	return nil
}

func (q *queueTestDouble) Receive(_ context.Context, _ int32, _ int32) ([]types.Message, error) {
	if atomic.AddInt32(&q.receiveCount, 1) == 1 {
		return []types.Message{
			{
				Body:          aws.String(`{"body":"payload","attributes":{"type":"message"}}`),
				MessageId:     aws.String("message-id"),
				ReceiptHandle: aws.String("receipt-handle"),
			},
		}, nil
	}

	time.Sleep(5 * time.Millisecond)
	return []types.Message{}, nil
}

func (q *queueTestDouble) Send(context.Context, *sqs.Message) error {
	return nil
}

func (q *queueTestDouble) SendBatch(context.Context, []*sqs.Message) error {
	return nil
}
