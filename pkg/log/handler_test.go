package log_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateFromConfig(t *testing.T) {
	config := cfg.New()
	if err := config.Option(cfg.WithConfigFile("testdata/config.yml", "yml")); err != nil {
		assert.FailNow(t, "can not load config: %s", err)
	}

	var err error
	var handlers []log.Handler

	if handlers, err = log.NewHandlersFromConfig(config); err != nil {
		assert.FailNow(t, "can not create logger: %s", err)
	}

	assert.Len(t, handlers, 1)
}
