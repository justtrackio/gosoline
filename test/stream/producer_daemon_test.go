// +build integration

package stream_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
	"time"
)

type TestData struct {
	Data string `json:"data"`
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

func (s *ProducerDaemonTestSuite) TearDownSuite() {
	if s.mocks != nil {
		s.mocks.Shutdown()
		s.mocks = nil
	}
}

func (s *ProducerDaemonTestSuite) SetupTest() {
	kinesisEndpoint := fmt.Sprintf("http://%s:%d", s.mocks.ProvideKinesisHost("kinesis"), s.mocks.ProvideKinesisPort("kinesis"))
	sqsEndpoint := fmt.Sprintf("http://%s:%d", s.mocks.ProvideSqsHost("sns_sqs"), s.mocks.ProvideSqsPort("sns_sqs"))

	err := os.Setenv("AWS_KINESIS_ENDPOINT", kinesisEndpoint)
	s.NoError(err)
	err = os.Setenv("AWS_SQS_ENDPOINT", sqsEndpoint)
	s.NoError(err)
}

func (s *ProducerDaemonTestSuite) TestWriteData() {
	args := os.Args
	os.Args = args[:1]
	defer func() {
		os.Args = args
	}()

	app := application.Default(application.WithLoggerHook(s))
	app.Add("testModule", &testModule{})
	app.Run()
}

func (s *ProducerDaemonTestSuite) Fire(level string, msg string, err error, data *mon.Metadata) error {
	s.NoError(err)
	s.Contains([]string{"debug", "info", "warn"}, level, "Unexpected log message: [%s] %s %v %v", level, msg, err, data)

	return nil
}

type testModule struct {
	kernel.EssentialModule

	producerKinesis stream.Producer
	producerSqs     stream.Producer
}

func (m *testModule) Boot(config cfg.Config, logger mon.Logger) error {
	m.producerKinesis = stream.NewProducer(config, logger, "testDataKinesis")
	m.producerSqs = stream.NewProducer(config, logger, "testDataSqs")

	return nil
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

func TestProducerDaemon(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
