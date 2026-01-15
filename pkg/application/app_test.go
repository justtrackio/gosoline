package application_test

import (
	"context"
	"os"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

type testSettings struct {
	Field string `cfg:"field" default:"def"`
}

type testModule struct {
	kernel.EssentialModule
	t *testing.T
}

func (m testModule) Boot(config cfg.Config, _ log.Logger) error {
	settings := &testSettings{}
	if err := config.UnmarshalKey("test.settings-struct", settings); err != nil {
		return err
	}

	assert.Equal(m.t, "value", settings.Field)

	return nil
}

func (m testModule) Run(_ context.Context) error {
	return nil
}

func TestDefaultConfigParser(t *testing.T) {
	t.Setenv("TEST_SETTINGS_STRUCT_FIELD", "value")

	runTestApp(t, func() {
		config := application.WithConfigFile("config.dist.yml", "yml")
		exitCodeHandler := application.WithKernelExitHandler(func(code int) {})
		moduleOption := application.WithModuleFactory("test", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return testModule{
				t: t,
			}, nil
		})

		app := application.Default(config, exitCodeHandler, moduleOption)
		app.Run()
	})
}

func runTestApp(t *testing.T, f func()) {
	oldDir, err := os.Getwd()
	assert.NoError(t, err)

	t.Chdir("testdata")

	defer func() {
		t.Chdir(oldDir)
	}()

	args := os.Args
	os.Args = []string{os.Args[0]}
	defer func() {
		os.Args = args
		assert.Nil(t, recover(), "App should not fail to be created")
	}()

	f()
}
