package aws_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/stretchr/testify/suite"
)

func TestRealmTestSuite(t *testing.T) {
	suite.Run(t, new(RealmTestSuite))
}

type RealmTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	appId       cfg.AppId
}

func (s *RealmTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.appId = cfg.AppId{
		Project:     "justtrack",
		Environment: "test",
		Family:      "gosoline",
		Group:       "group",
		Application: "producer",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *RealmTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *RealmTestSuite) TestRealmDefault() {
	// Test default realm pattern
	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "default")
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group", realm)
}

func (s *RealmTestSuite) TestRealmGlobalCustomPattern() {
	// Test custom global realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern": "{project}-{env}-{family}",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "default")
	s.NoError(err)
	s.Equal("justtrack-test-gosoline", realm)
}

func (s *RealmTestSuite) TestRealmServiceSpecificPattern() {
	// Test service-specific realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.realm.pattern": "{project}-{env}",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "default")
	s.NoError(err)
	s.Equal("justtrack-test", realm)
}

func (s *RealmTestSuite) TestRealmClientSpecificPattern() {
	// Test client-specific realm pattern
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.specific.naming.realm.pattern": "{project}-{family}",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "specific")
	s.NoError(err)
	s.Equal("justtrack-gosoline", realm)
}

func (s *RealmTestSuite) TestRealmClientSpecificWithFallback() {
	// Test client-specific fallback to service default realm
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.realm.pattern": "{project}-{env}",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "specific")
	s.NoError(err)
	s.Equal("justtrack-test", realm)
}

func (s *RealmTestSuite) TestRealmWithAppField() {
	// Test realm pattern with app field
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern": "{project}-{app}",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "default")
	s.NoError(err)
	s.Equal("justtrack-producer", realm)
}

func (s *RealmTestSuite) TestRealmPriorityOrder() {
	// Test that client-specific config takes priority over service default and global
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern":                             "{project}-GLOBAL",
		"cloud.aws.sqs.clients.default.naming.realm.pattern": "{project}-DEFAULT",
		"cloud.aws.sqs.clients.specific.naming.realm.pattern": "{project}-SPECIFIC",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "specific")
	s.NoError(err)
	s.Equal("justtrack-SPECIFIC", realm)
}

func (s *RealmTestSuite) TestRealmFallbackToDefault() {
	// Test fallback from client-specific to service default
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern":                             "{project}-GLOBAL",
		"cloud.aws.sqs.clients.default.naming.realm.pattern": "{project}-DEFAULT",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "specific")
	s.NoError(err)
	s.Equal("justtrack-DEFAULT", realm)
}

func (s *RealmTestSuite) TestRealmFallbackToGlobal() {
	// Test fallback from client-specific to global
	s.setupConfig(map[string]any{
		"cloud.aws.realm.pattern": "{project}-GLOBAL",
	})

	realm, err := aws.ResolveRealm(s.config, s.appId, "sqs", "specific")
	s.NoError(err)
	s.Equal("justtrack-GLOBAL", realm)
}