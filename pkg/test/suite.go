package test

import (
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type TestingSuite interface {
	Env() *env.Environment
	SetEnv(environment *env.Environment)
	SetT(t *testing.T)
	T() *testing.T
	SetupSuite() []SuiteOption
}

type Suite struct {
	*assert.Assertions
	require *require.Assertions
	t       *testing.T

	env *env.Environment
}

func (s *Suite) Env() *env.Environment {
	if s.env == nil {
		assert.FailNow(s.t, "test environment not running yet", "to setup your test environment, use the WithEnvSetup option instead of accessing the env directly")
	}

	return s.env
}

func (s *Suite) SetEnv(env *env.Environment) {
	s.env = env
}

func (s *Suite) SetT(t *testing.T) {
	s.t = t
	s.Assertions = assert.New(t)
	s.require = require.New(t)
}

func (s *Suite) T() *testing.T {
	return s.t
}
