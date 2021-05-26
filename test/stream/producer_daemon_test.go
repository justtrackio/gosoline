// +build integration

package stream_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/encoding/base64"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"os"
	"testing"
	"time"
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

	err := os.Setenv("AWS_KINESIS_ENDPOINT", kinesisEndpoint)
	s.NoError(err)
	err = os.Setenv("AWS_SQS_ENDPOINT", sqsEndpoint)
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

	app := application.Default(application.WithLoggerHook(s))
	app.Add("testModule", newTestModule)
	app.Add("testCompressionModule", newTestCompressionModule)
	app.Run()
}

func (s *ProducerDaemonTestSuite) Fire(level string, msg string, err error, data *mon.Metadata) error {
	s.NoError(err)
	s.Contains([]string{"debug", "info", "warn"}, level, "Unexpected log message: [%s] %s %v %v", level, msg, err, data)

	return nil
}

type testModule struct {
	kernel.ForegroundModule

	producerKinesis stream.Producer
	producerSqs     stream.Producer
}

func newTestModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	var err error
	var kinesisProducer, sqsProducer stream.Producer

	if kinesisProducer, err = stream.NewProducer(config, logger, "testDataKinesis"); err != nil {
		return nil, err
	}

	if sqsProducer, err = stream.NewProducer(config, logger, "testDataSqs"); err != nil {
		return nil, err
	}

	module := &testModule{
		producerKinesis: kinesisProducer,
		producerSqs:     sqsProducer,
	}

	return module, nil
}

func (m *testModule) Run(ctx context.Context) error {
	time.Sleep(time.Second)

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

	logger      mon.Logger
	producerSqs stream.Producer
	inputSqs    stream.Input
}

func newTestCompressionModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	var err error
	var sqsProducer stream.Producer
	var sqsInput stream.Input

	if sqsProducer, err = stream.NewProducer(config, logger, "testEventsSqs"); err != nil {
		return nil, err
	}

	if sqsInput, err = stream.NewConfigurableInput(config, logger, "testEventsSqs"); err != nil {
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

				m.logger.Infof("Message attributes: %v", msg.Attributes)
				m.logger.Infof("Message encoded body: %s", msg.Body)
				m.logger.Infof("Message decoded body: %s", string(body))
			case <-ctx.Done():
				return multiErr
			}
		}
	})

	return cfn.Wait()
}

func TestProducerDaemon(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
