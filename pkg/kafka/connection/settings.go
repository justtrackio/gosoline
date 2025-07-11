package connection

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type Settings struct {
	// Connection.
	Bootstrap          []string `cfg:"bootstrap" validate:"required"`
	InsecureSkipVerify bool     `cfg:"insecure_skip_verify"`
	TlsEnabled         bool     `cfg:"tls_enabled" default:"true"`

	// Credentials.
	Username string `cfg:"username"`
	Password string `cfg:"password"`
}

func ParseSettings(c cfg.Config, name string) (*Settings, error) {
	settings := &Settings{}
	key := fmt.Sprintf("kafka.connection.%s", name)
	if err := c.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka connection settings for key %q in ParseSettings: %w", key, err)
	}

	return settings, nil
}
