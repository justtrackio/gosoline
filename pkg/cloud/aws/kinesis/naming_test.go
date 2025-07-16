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
	s.settings = &kinesis.Settings{
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "env",
			Family:      "gosoline",
			Group:       "grp",
			Application: "producer",
			Realm:       "justtrack-env-gosoline-grp",
		},
		ClientName: "default",
		StreamName: "event",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *GetStreamNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
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
	// Test custom global realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern": "{project}-{env}-{family}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmServiceSpecificPattern() {
	// Test service-specific realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.realm.pattern": "{project}-{env}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmClientSpecificPattern() {
	// Test client-specific realm pattern
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.specific.naming.realm.pattern": "{project}-{family}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-gosoline-event", name)
}

func (s *GetStreamNameTestSuite) TestRealmWithCustomPattern() {
	// Test custom pattern with realm
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern":                           "{project}-{env}-{family}",
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
