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

func ParseSettings(c cfg.Config, name string) *Settings {
	settings := &Settings{}
	c.UnmarshalKey(fmt.Sprintf("kafka.connection.%s", name), settings)

	return settings
}
