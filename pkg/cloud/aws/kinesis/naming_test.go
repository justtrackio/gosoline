package kinesis_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/stretchr/testify/suite"
)

func TestGetStreamNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetStreamNameTestSuite))
}

type GetStreamNameTestSuite struct {
	suite.Suite
	config   cfg.GosoConf
	settings *kinesis.Settings
}

func (s *GetStreamNameTestSuite) SetupTest() {
	s.config = cfg.New()
	s.settings = &kinesis.Settings{
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "env",
			Family:      "gosoline",
			Group:       "grp",
			Application: "producer",
		},
		ClientName: "default",
		StreamName: "event",
	}
}

func (s *GetStreamNameTestSuite) setupConfig(settings map[string]interface{}) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetStreamNameTestSuite) TestDefault() {
	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-grp-event", string(name))
}

func (s *GetStreamNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]interface{}{
		"cloud.aws.kinesis.clients.default.naming.pattern": "{app}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]interface{}{
		"cloud.aws.kinesis.clients.specific.naming.pattern": "{app}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}
