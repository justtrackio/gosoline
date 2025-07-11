package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
)

func TestOsPrefixExists(t *testing.T) {
	t.Setenv("GOSOLINE_TEST", "foobar")

	provider := cfg.NewOsEnvProvider()
	assert.True(t, provider.PrefixExists("GOSOLINE"))
}

func TestMemoryPrefixExists(t *testing.T) {
	provider := cfg.NewMemoryEnvProvider(map[string]string{
		"GOSOLINE_TEST": "foobar",
	})
	assert.True(t, provider.PrefixExists("GOSOLINE"))
}
