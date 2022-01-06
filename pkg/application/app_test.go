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
	config.UnmarshalKey("test.settings-struct", settings)

	assert.Equal(m.t, "value", settings.Field)

	return nil
}

func (m testModule) Run(_ context.Context) error {
	return nil
}

func TestDefaultConfigParser(t *testing.T) {
	err := os.Setenv("TEST_SETTINGS_STRUCT_FIELD", "value")
	assert.NoError(t, err)

	runTestApp(t, func() {
		exitCodeHandler := application.WithKernelExitHandler(func(code int) {})

		app := application.Default(exitCodeHandler)
		app.Add("test", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return testModule{
				t: t,
			}, nil
		})
		app.Run()
	})
}

func runTestApp(t *testing.T, f func()) {
	oldDir, err := os.Getwd()
	assert.NoError(t, err)

	err = os.Chdir("testdata")
	assert.NoError(t, err)

	defer func() {
		err := os.Chdir(oldDir)
		assert.NoError(t, err)
	}()

	args := os.Args
	os.Args = []string{os.Args[0]}
	defer func() {
		os.Args = args
	}()

	f()
}
