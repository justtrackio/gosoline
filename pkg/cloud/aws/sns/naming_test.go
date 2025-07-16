package sns_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/stretchr/testify/suite"
)

func TestGetTopicNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetTopicNameTestSuite))
}

type GetTopicNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    sns.TopicNameSettings
}

func (s *GetTopicNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	
	// Set up basic config values
	baseConfig := map[string]any{
		"app_project": "justtrack",
		"env":         "test",
		"app_family":  "gosoline",
		"app_group":   "group",
		"app_name":    "producer",
		"realm":       "justtrack-test-gosoline-group", // Default realm value
	}
	
	err := s.config.Option(cfg.WithConfigMap(baseConfig))
	s.NoError(err)
	
	err = s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
	
	// Create AppId from config
	appId, err := cfg.GetAppIdFromConfig(s.config)
	s.NoError(err)
	
	s.settings = sns.TopicNameSettings{
		AppId:      appId,
		ClientName: "default",
		TopicId:    "event",
	}
}

func (s *GetTopicNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
	
	// Recreate AppId from config to pick up new configuration
	appId, err := cfg.GetAppIdFromConfig(s.config)
	s.NoError(err)
	s.settings.AppId = appId
}

func (s *GetTopicNameTestSuite) TestDefault() {
	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *GetTopicNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetTopicNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.pattern": "{app}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.specific.naming.pattern": "{app}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.pattern": "{app}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_SNS_CLIENTS_SPECIFIC_NAMING_PATTERN": "!nodecode {app}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestRealmDefault() {
	// Test default realm pattern resolves correctly
	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *GetTopicNameTestSuite) TestRealmGlobalCustomPattern() {
	// Test custom global realm
	s.setupConfig(map[string]any{
		"realm": "justtrack-test-gosoline",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-event", name)
}

func (s *GetTopicNameTestSuite) TestRealmServiceSpecificPattern() {
	// Test service-specific realm
	s.setupConfig(map[string]any{
		"realm": "justtrack-test",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-event", name)
}

func (s *GetTopicNameTestSuite) TestRealmClientSpecificPattern() {
	// Test client-specific realm
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"realm": "justtrack-gosoline",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-gosoline-event", name)
}

func (s *GetTopicNameTestSuite) TestRealmWithCustomPattern() {
	// Test custom pattern with realm
	s.setupConfig(map[string]any{
		"realm": "justtrack-test-gosoline",
		"cloud.aws.sns.clients.default.naming.pattern": "{realm}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-event", name)
}

func (s *GetTopicNameTestSuite) TestBackwardCompatibilityWithoutRealm() {
	// Test that old patterns still work without realm
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.pattern": "{project}-{env}-{family}-{group}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}
