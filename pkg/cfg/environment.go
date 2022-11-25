package cfg

import (
	"os"
)

type EnvProvider interface {
	LookupEnv(key string) (string, bool)
	SetEnv(key string, value string) error
}

type osEnvProvider struct{}

func NewOsEnvProvider() *osEnvProvider {
	return &osEnvProvider{}
}

func (o *osEnvProvider) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (o *osEnvProvider) SetEnv(key string, value string) error {
	return os.Setenv(key, value)
}

type memoryEnvProvider struct {
	values map[string]string
}

func NewMemoryEnvProvider() *memoryEnvProvider {
	return &memoryEnvProvider{
		values: make(map[string]string),
	}
}

func (m *memoryEnvProvider) LookupEnv(key string) (string, bool) {
	val, ok := m.values[key]

	return val, ok
}

func (m *memoryEnvProvider) SetEnv(key string, value string) error {
	m.values[key] = value
	return nil
}
