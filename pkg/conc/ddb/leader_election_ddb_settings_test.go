package ddb_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	concDdb "github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/stretchr/testify/suite"
)

type ReadLeaderElectionDdbSettingsTestSuite struct {
	suite.Suite

	config cfg.GosoConf
}

func (s *ReadLeaderElectionDdbSettingsTestSuite) SetupTest() {
	s.config = cfg.New()
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "my-app",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
			},
		},
	}))
	s.Require().NoError(err, "base config creation should not fail")
}

func (s *ReadLeaderElectionDdbSettingsTestSuite) TestDefaultValues() {
	settings, err := concDdb.ReadLeaderElectionDdbSettings(s.config, "my-election")

	s.NoError(err)
	s.Require().NotNil(settings)
	s.Equal("{app.namespace}-leader-elections", settings.Naming.TablePattern)
	s.Equal("-", settings.Naming.TableDelimiter)
	s.Equal("default", settings.ClientName)
	s.Equal("my-app", settings.GroupId)
	s.Equal(time.Minute, settings.LeaseDuration)
}

func (s *ReadLeaderElectionDdbSettingsTestSuite) TestExplicitValues() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"conc.leader_election.my-election": map[string]any{
			"naming": map[string]any{
				"table_pattern":   "custom-leader-elections",
				"table_delimiter": "_",
			},
			"client_name":    "custom-client",
			"group_id":       "my-group",
			"lease_duration": "5m",
		},
	}))
	s.Require().NoError(err)

	settings, err := concDdb.ReadLeaderElectionDdbSettings(s.config, "my-election")

	s.NoError(err)
	s.Require().NotNil(settings)
	s.Equal("custom-leader-elections", settings.Naming.TablePattern)
	s.Equal("_", settings.Naming.TableDelimiter)
	s.Equal("custom-client", settings.ClientName)
	s.Equal("my-group", settings.GroupId)
	s.Equal(5*time.Minute, settings.LeaseDuration)
}

func (s *ReadLeaderElectionDdbSettingsTestSuite) TestDefaultsInheritedFromDefaultKey() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"conc.leader_election.default": map[string]any{
			"client_name":    "shared-client",
			"lease_duration": "2m",
		},
		"conc.leader_election.my-election": map[string]any{
			"group_id": "my-group",
		},
	}))
	s.Require().NoError(err)

	settings, err := concDdb.ReadLeaderElectionDdbSettings(s.config, "my-election")

	s.NoError(err)
	s.Require().NotNil(settings)
	s.Equal("shared-client", settings.ClientName)
	s.Equal(2*time.Minute, settings.LeaseDuration)
	s.Equal("my-group", settings.GroupId)
}

func (s *ReadLeaderElectionDdbSettingsTestSuite) TestExplicitValuesOverrideDefaults() {
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"conc.leader_election.default": map[string]any{
			"client_name":    "shared-client",
			"lease_duration": "2m",
		},
		"conc.leader_election.my-election": map[string]any{
			"client_name":    "override-client",
			"lease_duration": "30s",
		},
	}))
	s.Require().NoError(err)

	settings, err := concDdb.ReadLeaderElectionDdbSettings(s.config, "my-election")

	s.NoError(err)
	s.Require().NotNil(settings)
	s.Equal("override-client", settings.ClientName)
	s.Equal(30*time.Second, settings.LeaseDuration)
}

func TestReadLeaderElectionDdbSettingsTestSuite(t *testing.T) {
	suite.Run(t, new(ReadLeaderElectionDdbSettingsTestSuite))
}
