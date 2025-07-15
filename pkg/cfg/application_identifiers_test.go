package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestGetAppIdFromConfig(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app_project").Return("prj", nil)
	config.EXPECT().GetString("app_family").Return("fam", nil)
	config.EXPECT().GetString("app_group").Return("grp", nil)
	config.EXPECT().GetString("app_name").Return("name", nil)
	config.EXPECT().GetString("env").Return("test", nil)

	appId, err := cfg.GetAppIdFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
		Realm:       "",
	}, appId)
}

func TestAppId_PadFromConfig(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app_project").Return("prj", nil)
	config.EXPECT().GetString("app_family").Return("fam", nil)
	config.EXPECT().GetString("app_group").Return("grp", nil)
	config.EXPECT().GetString("app_name").Return("name", nil)
	config.EXPECT().GetString("env").Return("test", nil)

	appId := cfg.AppId{}
	err := appId.PadFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
		Realm:       "",
	}, appId)

	config.AssertExpectations(t)
}

func TestAppId_ReplaceMacros(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
	}

	pattern := "{project}-{env}-{family}-{group}-{app}"
	result := appId.ReplaceMacros(pattern)
	assert.Equal(t, "myproject-test-myfamily-mygroup-myapp", result)
}

func TestAppId_ReplaceMacros_EmptyValues(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "",
		Family:      "myfamily",
		Group:       "",
		Application: "myapp",
	}

	pattern := "{project}-{env}-{family}-{group}-{app}"
	result := appId.ReplaceMacros(pattern)
	assert.Equal(t, "myproject--myfamily--myapp", result)
}

func TestAppId_ReplaceMacros_WithRealm(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "myproject-test",
	}

	pattern := "{realm}-{streamName}"
	extraMacros := []cfg.MacroValue{
		{"streamName", "mystream"},
	}
	result := appId.ReplaceMacros(pattern, extraMacros...)
	assert.Equal(t, "myproject-test-mystream", result)
}

func TestAppId_ReplaceMacros_ExtraMacrosOrdering(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
	}

	// Test that extra macros are replaced before and after AppId macros
	pattern := "{prefix}-{project}-{suffix}"
	extraMacros := []cfg.MacroValue{
		{"prefix", "before-{env}"},
		{"suffix", "after-{env}"},
	}
	result := appId.ReplaceMacros(pattern, extraMacros...)
	assert.Equal(t, "before-test-myproject-after-test", result)
}

type RealmTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
}

func (s *RealmTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *RealmTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *RealmTestSuite) TestResolveRealm_Default() {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "",
	}

	realm, err := cfg.ResolveRealm(s.config, appId, "kinesis", "default")
	s.NoError(err)
	s.Equal("myproject-test-myfamily-mygroup", realm)
}

func (s *RealmTestSuite) TestResolveRealm_GlobalCustomPattern() {
	s.setupConfig(map[string]any{
		"cloud": map[string]any{
			"aws": map[string]any{
				"realm": map[string]any{
					"pattern": "custom-{project}-{env}",
				},
			},
		},
	})

	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "",
	}

	realm, err := cfg.ResolveRealm(s.config, appId, "kinesis", "default")
	s.NoError(err)
	s.Equal("custom-myproject-test", realm)
}

func (s *RealmTestSuite) TestResolveRealm_ServiceSpecificPattern() {
	s.setupConfig(map[string]any{
		"cloud": map[string]any{
			"aws": map[string]any{
				"kinesis": map[string]any{
					"clients": map[string]any{
						"default": map[string]any{
							"naming": map[string]any{
								"realm": map[string]any{
									"pattern": "kinesis-{project}-{env}-{family}",
								},
							},
						},
					},
				},
			},
		},
	})

	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "",
	}

	realm, err := cfg.ResolveRealm(s.config, appId, "kinesis", "default")
	s.NoError(err)
	s.Equal("kinesis-myproject-test-myfamily", realm)
}

func (s *RealmTestSuite) TestResolveRealm_ClientSpecificPattern() {
	s.setupConfig(map[string]any{
		"cloud": map[string]any{
			"aws": map[string]any{
				"kinesis": map[string]any{
					"clients": map[string]any{
						"specific": map[string]any{
							"naming": map[string]any{
								"realm": map[string]any{
									"pattern": "specific-{project}-{env}-{app}",
								},
							},
						},
					},
				},
			},
		},
	})

	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "",
	}

	realm, err := cfg.ResolveRealm(s.config, appId, "kinesis", "specific")
	s.NoError(err)
	s.Equal("specific-myproject-test-myapp", realm)
}

func TestRealmTestSuite(t *testing.T) {
	suite.Run(t, new(RealmTestSuite))
}
