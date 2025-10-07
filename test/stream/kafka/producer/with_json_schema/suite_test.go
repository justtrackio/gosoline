//go:build integration && fixtures

package with_json_schema

import (
	_ "embed"
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/test/stream/kafka"
	"github.com/justtrackio/gosoline/test/stream/kafka/producer"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
	"github.com/twmb/franz-go/pkg/sr"
)

type testSuite struct {
	suite.Suite
	produceCount int
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() []suite.Option {
	s.produceCount = 3
	schemaSettings := stream.SchemaSettings{
		Subject: "testEvent",
		Schema:  testEvent.SchemaJson,
		Model:   &testEvent.TestEvent{},
	}

	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithConfigFile("../config.dist.yml"),
		kafka.WithKafkaBrokerPort(9196),
		suite.WithModule("producer-module", producer.NewProducerModule(s.produceCount, stream.WithSchemaSettings(schemaSettings))),
		kafka.WithRegisteredSchema(s, schemaSettings.Subject, testEvent.SchemaJson, sr.TypeJSON),
	}
}

func (s *testSuite) TestProduce(app suite.AppUnderTest) {
	producer.CheckExpectedKafkaEndOffset(s, app, int64(s.produceCount))
}
