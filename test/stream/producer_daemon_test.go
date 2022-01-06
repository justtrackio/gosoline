//go:build integration
// +build integration

package stream_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	pkgTest "github.com/justtrackio/gosoline/pkg/test"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestData struct {
	Data string `json:"data"`
}

type TestEvent struct {
	Id         string    `json:"id"`
	Name       string    `json:"name"`
	HappenedAt time.Time `json:"happened_at"`
}

type TestEventsBatch struct {
	UserId string      `json:"user_id"`
	Events []TestEvent `json:"events"`
}

type LoggingHandler struct {
	t *testing.T
}

func (l LoggingHandler) Channels() []string {
	return []string{}
}

func (l LoggingHandler) Level() int {
	return log.PriorityDebug
}

func (l LoggingHandler) Log(_ time.Time, level int, msg string, _ []interface{}, err error, data log.Data) error {
	assert.NoError(l.t, err)
	assert.Contains(l.t, []int{log.PriorityDebug, log.PriorityInfo, log.PriorityWarn}, level, "Unexpected log message: [%d] %s %v %v", level, msg, err, data)
	return nil
}

type ProducerDaemonTestSuite struct {
	suite.Suite
	mocks *pkgTest.Mocks
}

func (s *ProducerDaemonTestSuite) SetupSuite() {
	err := os.Setenv("AWS_ACCESS_KEY_ID", "a")
	s.NoError(err)
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "b")
	s.NoError(err)

	mocks, err := pkgTest.Boot("../test_configs/config.kinesis.test.yml", "../test_configs/config.sns_sqs.test.yml")
	if err != nil {
		if mocks != nil {
			mocks.Shutdown()
		}

		s.Fail("failed to boot mocks: %s", err.Error())

		return
	}

	s.mocks = mocks
}

func (s *ProducerDaemonTestSuite) SetupTest() {
	kinesisEndpoint := fmt.Sprintf("http://%s:%d", s.mocks.ProvideKinesisHost("kinesis"), s.mocks.ProvideKinesisPort("kinesis"))
	sqsEndpoint := fmt.Sprintf("http://%s:%d", s.mocks.ProvideSqsHost("sns_sqs"), s.mocks.ProvideSqsPort("sns_sqs"))

	err := os.Setenv("CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_ENDPOINT", kinesisEndpoint)
	s.NoError(err)
	err = os.Setenv("CLOUD_AWS_SQS_CLIENTS_DEFAULT_ENDPOINT", sqsEndpoint)
	s.NoError(err)
	// TODO: remove once https://github.com/justtrackio/gosoline/issues/649 has been solved and we no longer need this work around
	err = os.Setenv("CLOUD_AWS_SQS_CLIENTS_STREAM_OUTPUT_TESTDATAFIFOSQS_ENDPOINT", sqsEndpoint)
	s.NoError(err)
	err = os.Setenv("CLOUD_AWS_SQS_CLIENTS_STREAM_OUTPUT_TESTEVENTSSQS_ENDPOINT", sqsEndpoint)
	s.NoError(err)
}

func (s *ProducerDaemonTestSuite) TearDownSuite() {
	if s.mocks != nil {
		s.mocks.Shutdown()
		s.mocks = nil
	}
}

func (s *ProducerDaemonTestSuite) TestWriteData() {
	args := os.Args
	os.Args = args[:1]
	defer func() {
		os.Args = args
	}()

	handler := LoggingHandler{
		t: s.T(),
	}

	app := application.Default(application.WithLoggerHandlers(handler), application.WithKernelExitHandler(func(code int) {
		assert.Equal(s.T(), kernel.ExitCodeOk, code, "exit code should be %d", kernel.ExitCodeOk)
	}))
	app.Add("testModule", newTestModule)
	app.Add("testCompressionModule", newTestCompressionModule)
	app.Add("testFifoModule", newTestFifoModule(s.T()))
	app.Run()
}

type testModule struct {
	kernel.ForegroundModule

	producerKinesis stream.Producer
	producerSqs     stream.Producer
}

func newTestModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var err error
	var kinesisProducer, sqsProducer stream.Producer

	if kinesisProducer, err = stream.NewProducer(ctx, config, logger, "testDataKinesis"); err != nil {
		return nil, err
	}

	if sqsProducer, err = stream.NewProducer(ctx, config, logger, "testDataSqs"); err != nil {
		return nil, err
	}

	module := &testModule{
		producerKinesis: kinesisProducer,
		producerSqs:     sqsProducer,
	}

	return module, nil
}

func (m *testModule) Run(ctx context.Context) error {
	for i := 0; i < 13; i++ {
		if err := m.producerKinesis.WriteOne(ctx, &TestData{}); err != nil {
			return err
		}

		if err := m.producerSqs.WriteOne(ctx, &TestData{}); err != nil {
			return err
		}
	}

	return nil
}

type testCompressionModule struct {
	kernel.ForegroundModule

	logger      log.Logger
	producerSqs stream.Producer
	inputSqs    stream.Input
}

func newTestCompressionModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	logger = logger.WithChannel("compression_module")
	var err error
	var sqsProducer stream.Producer
	var sqsInput stream.Input

	if sqsProducer, err = stream.NewProducer(ctx, config, logger, "testEventsSqs"); err != nil {
		return nil, err
	}

	if sqsInput, err = stream.NewConfigurableInput(ctx, config, logger, "testEventsSqs"); err != nil {
		return nil, err
	}

	module := &testCompressionModule{
		logger:      logger,
		producerSqs: sqsProducer,
		inputSqs:    sqsInput,
	}

	return module, nil
}

func (m *testCompressionModule) Run(ctx context.Context) error {
	for i := 0; i < 8; i++ {
		time.Sleep(time.Millisecond * 10)

		now := time.Now().UTC()
		event := &TestEventsBatch{
			UserId: uuid.New().NewV4(),
			Events: []TestEvent{
				{
					Id:         uuid.New().NewV4(),
					Name:       "session_tracking_start",
					HappenedAt: now,
				},
				{
					Id:         uuid.New().NewV4(),
					Name:       "session_app_open",
					HappenedAt: now.Add(time.Millisecond),
				},
				{
					Id:         uuid.New().NewV4(),
					Name:       "session_tracking_end",
					HappenedAt: now.Add(time.Minute),
				},
			},
		}
		if err := m.producerSqs.WriteOne(ctx, event); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*20))
	cfn := coffin.New()
	cfn.GoWithContext(ctx, m.inputSqs.Run)
	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		defer m.inputSqs.Stop()
		defer cancel()

		var multiErr error
		for {
			select {
			case msg := <-m.inputSqs.Data():
				if msg.Attributes[stream.AttributeEncoding] != string(stream.EncodingJson) {
					multiErr = multierror.Append(multiErr, fmt.Errorf("unexpected encoding, expected %q, got %q", stream.EncodingJson, msg.Attributes[stream.AttributeEncoding]))
					continue
				}

				if msg.Attributes[stream.AttributeCompression] != string(stream.CompressionGZip) {
					multiErr = multierror.Append(multiErr, fmt.Errorf("unexpected compression, expected %q, got %q", stream.CompressionGZip, msg.Attributes[stream.AttributeCompression]))
					continue
				}

				decoded, err := base64.DecodeString(msg.Body)
				if err != nil {
					multiErr = multierror.Append(multiErr, fmt.Errorf("failed to decode message body from base64: %w", err))
					continue
				}

				reader, err := gzip.NewReader(bytes.NewReader(decoded))
				if err != nil {
					multiErr = multierror.Append(multiErr, fmt.Errorf("failed to create gzip reader: %w", err))
					continue
				}

				body, err := ioutil.ReadAll(reader)
				if err != nil {
					multiErr = multierror.Append(multiErr, fmt.Errorf("failed to consume gzip reader: %w", err))
					continue
				}

				m.logger.Info("Message attributes: %v", msg.Attributes)
				m.logger.Info("Message encoded body: %s", msg.Body)
				m.logger.Info("Message decoded body: %s", string(body))
			case <-ctx.Done():
				return multiErr
			}
		}
	})

	return cfn.Wait()
}

type testFifoModule struct {
	kernel.ForegroundModule

	producerFifoSqs stream.Producer
	consumer        kernel.Module
	cancel          func()
	assert          func()
}

type testFifoConsumerCallback struct {
	callback func(data *TestData, attributes map[string]interface{})
}

func (t testFifoConsumerCallback) GetModel(_ map[string]interface{}) interface{} {
	return &TestData{}
}

func (t testFifoConsumerCallback) Consume(_ context.Context, model interface{}, attributes map[string]interface{}) (bool, error) {
	t.callback(model.(*TestData), attributes)

	return true, nil
}

func newTestFifoModule(t *testing.T) func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		logger = logger.WithChannel("fifo_module")
		var err error
		var sqsProducer stream.Producer

		if sqsProducer, err = stream.NewProducer(ctx, config, logger, "testDataFifoSqs"); err != nil {
			return nil, err
		}

		var lck sync.Mutex
		receivedMessages := make(map[string]bool)
		calls := 0

		module := &testFifoModule{
			producerFifoSqs: sqsProducer,
			assert: func() {
				assert.Equal(t, 8, calls)
				assert.Equal(t, map[string]bool{
					"0": true,
					"1": true,
					"2": true,
					"3": true,
					"4": true,
					"5": true,
					"6": true,
					"7": true,
				}, receivedMessages)
			},
		}

		factory := stream.NewConsumer("testDataFifoSqs", func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback, error) {
			return testFifoConsumerCallback{
				func(data *TestData, attributes map[string]interface{}) {
					lck.Lock()
					defer lck.Unlock()

					logger.WithContext(ctx).Info("Got message with body %s", data.Data)

					assert.Equal(t, "my_value", attributes["my_attribute"])
					assert.Contains(t, []string{"0", "1", "2", "3"}, attributes[sqs.AttributeSqsMessageGroupId])
					receivedMessages[attributes[sqs.AttributeSqsMessageDeduplicationId].(string)] = true
					calls++

					if calls == 8 {
						module.cancel()
					}
				},
			}, nil
		})

		if module.consumer, err = factory(ctx, config, logger); err != nil {
			return nil, err
		}

		return module, nil
	}
}

func (m *testFifoModule) Run(ctx context.Context) error {
	for i := 0; i < 14; i++ {
		if err := m.producerFifoSqs.WriteOne(ctx, &TestData{
			Data: fmt.Sprintf("%d", i),
		}, map[string]interface{}{
			"my_attribute":                         "my_value",
			sqs.AttributeSqsMessageGroupId:         fmt.Sprintf("%d", i%4),
			sqs.AttributeSqsMessageDeduplicationId: fmt.Sprintf("%d", i%8),
		}); err != nil {
			return err
		}
	}

	cancelContext, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	defer m.assert()

	return m.consumer.Run(cancelContext)
}

func TestProducerDaemon(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
