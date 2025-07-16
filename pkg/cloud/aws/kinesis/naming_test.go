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
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    *kinesis.Settings
}

func (s *GetStreamNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	
	// Set up basic config values
	baseConfig := map[string]any{
		"app_project": "justtrack",
		"env":         "env",
		"app_family":  "gosoline",
		"app_group":   "grp",
		"app_name":    "producer",
		"realm":       "justtrack-env-gosoline-grp", // Default realm value
	}
	
	err := s.config.Option(cfg.WithConfigMap(baseConfig))
	s.NoError(err)
	
	err = s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
	
	// Create AppId from config
	appId, err := cfg.GetAppIdFromConfig(s.config)
	s.NoError(err)
	
	s.settings = &kinesis.Settings{
		AppId:      appId,
		ClientName: "default",
		StreamName: "event",
	}
}

func (s *GetStreamNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
	
	// Recreate AppId from config to pick up new configuration
	appId, err := cfg.GetAppIdFromConfig(s.config)
	s.NoError(err)
	s.settings.AppId = appId
}

func (s *GetStreamNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetStreamNameTestSuite) TestDefault() {
	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-grp-event", string(name))
}

func (s *GetStreamNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.pattern": "{app}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.specific.naming.pattern": "{app}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.pattern": "{app}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_PATTERN": "!nodecode {app}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmDefault() {
	// Test default realm pattern resolves correctly
	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-grp-event", string(name))
}

func (s *GetStreamNameTestSuite) TestRealmGlobalCustomPattern() {
	// Test custom global realm
	s.setupConfig(map[string]any{
		"realm": "justtrack-env-gosoline",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmServiceSpecificPattern() {
	// Test service-specific realm
	s.setupConfig(map[string]any{
		"realm": "justtrack-env",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmClientSpecificPattern() {
	// Test client-specific realm
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"realm": "justtrack-gosoline",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-gosoline-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmWithCustomPattern() {
	// Test custom pattern with realm
	s.setupConfig(map[string]any{
		"realm": "justtrack-env-gosoline",
		"cloud.aws.kinesis.clients.default.naming.pattern": "{realm}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-event", name)
}

func (s *GetStreamNameTestSuite) TestBackwardCompatibilityWithoutRealm() {
	// Test that old patterns still work without realm
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.pattern": "{project}-{env}-{family}-{group}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-grp-event", name)
}
