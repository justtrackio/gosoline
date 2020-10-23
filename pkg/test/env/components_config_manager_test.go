package env_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ComponentsConfigManagerTestSuite struct {
	suite.Suite
	manager *env.ComponentsConfigManager
}

func (s *ComponentsConfigManagerTestSuite) SetupTest() {
	config := cfg.New()
	s.manager = env.NewComponentsConfigManager(config)
}

func (s *ComponentsConfigManagerTestSuite) TestAdd() {
	err := s.manager.Add(&env.ComponentBaseSettings{
		Name: "comp1Name",
		Type: "comp1Type",
	})
	s.NoError(err)

	err = s.manager.Add(&env.ComponentBaseSettings{
		Name: "comp2Name",
		Type: "comp2Type",
	})
	s.NoError(err)
}

func TestComponentsConfigManager(t *testing.T) {
	suite.Run(t, new(ComponentsConfigManagerTestSuite))
}
