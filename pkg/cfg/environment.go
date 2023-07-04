package cfg

import (
	"os"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

type EnvProvider interface {
	LookupEnv(key string) (string, bool)
	PrefixExists(prefix string) bool
	SetEnv(key string, value string) error
}

type osEnvProvider struct{}

func NewOsEnvProvider() *osEnvProvider {
	return &osEnvProvider{}
}

func (o *osEnvProvider) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (o *osEnvProvider) PrefixExists(prefix string) bool {
	envs := os.Environ()

	for _, env := range envs {
		key, _, _ := strings.Cut(env, "=")

		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

func (o *osEnvProvider) SetEnv(key string, value string) error {
	return os.Setenv(key, value)
}

type memoryEnvProvider struct {
	values map[string]string
}

func NewMemoryEnvProvider(initialValues ...map[string]string) *memoryEnvProvider {
	return &memoryEnvProvider{
		values: funk.MergeMaps(initialValues...),
	}
}

func (m *memoryEnvProvider) LookupEnv(key string) (string, bool) {
	val, ok := m.values[key]

	return val, ok
}

func (o *memoryEnvProvider) PrefixExists(prefix string) bool {
	for key := range o.values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

func (m *memoryEnvProvider) SetEnv(key string, value string) error {
	m.values[key] = value
	return nil
}
