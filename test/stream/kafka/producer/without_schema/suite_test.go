//go:build integration && fixtures

package without_schema

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/test/stream/kafka"
	"github.com/justtrackio/gosoline/test/stream/kafka/producer"
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

	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithConfigFile("../config.dist.yml"),
		kafka.WithKafkaBrokerPort(9291),
		suite.WithModule("producer-module", producer.NewProducerModule(s.produceCount)),
	}
}

func (s *testSuite) TestProduce(app suite.AppUnderTest) {
	producer.CheckExpectedKafkaEndOffset(s, app, int64(s.produceCount))
}
