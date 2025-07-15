package sqs_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/stretchr/testify/suite"
)

func TestGetSqsQueueNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetSqsQueueNameTestSuite))
}

type GetSqsQueueNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    sqs.QueueNameSettings
}

func (s *GetSqsQueueNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = sqs.QueueNameSettings{
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "group",
			Application: "producer",
		},
		ClientName: "default",
		QueueId:    "event",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *GetSqsQueueNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetSqsQueueNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetSqsQueueNameTestSuite) TestDefault() {
	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestDefaultFifo() {
	s.settings.FifoEnabled = true

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-producer-event.fifo", name)
}

func (s *GetSqsQueueNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.specific.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_PATTERN": "!nodecode {app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestRealmDefault() {
	// Test default realm pattern resolves correctly
	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestRealmGlobalCustomPattern() {
	// Test custom global realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern": "{project}-{env}-{family}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestRealmServiceSpecificPattern() {
	// Test service-specific realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.realm.pattern": "{project}-{env}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestRealmClientSpecificPattern() {
	// Test client-specific realm pattern
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.specific.naming.realm.pattern": "{project}-{family}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-gosoline-producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestRealmClientSpecificWithFallback() {
	// Test client-specific fallback to service default realm
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.realm.pattern": "{project}-{env}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestRealmWithCustomPattern() {
	// Test custom pattern with realm
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern":                       "{project}-{env}-{family}",
		"cloud.aws.sqs.clients.default.naming.pattern": "{realm}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestBackwardCompatibilityWithoutRealm() {
	// Test that old patterns still work without realm
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.pattern": "{project}-{env}-{family}-{group}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}
