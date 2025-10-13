package connection

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type Settings struct {
	// Connection.
	Brokers               []string      `cfg:"brokers" validate:"required"`
	SchemaRegistryAddress string        `cfg:"schema_registry_address"`
	InsecureSkipVerify    bool          `cfg:"insecure_skip_verify"`
	TlsEnabled            bool          `cfg:"tls_enabled" default:"true"`
	DialTimeout           time.Duration `cfg:"dial_timeout" default:"10s"`
	IsReadOnly            bool          `cfg:"is_read_only" default:"false"`

	// Credentials.
	Username string `cfg:"username"`
	Password string `cfg:"password"`
}

func ParseSettings(config cfg.Config, name string) (*Settings, error) {
	settings := &Settings{}
	key := fmt.Sprintf("kafka.connection.%s", name)
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka connection settings for key %q in ParseSettings: %w", key, err)
	}

	return settings, nil
}

func BuildConnectionOptions(config cfg.Config, connectionName string) ([]kgo.Opt, error) {
	conn, err := ParseSettings(config, connectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", connectionName, err)
	}

	options := []kgo.Opt{
		kgo.SeedBrokers(conn.Brokers...),
		kgo.DialTimeout(conn.DialTimeout),
	}

	if conn.Username != "" && conn.Password != "" {
		auth := scram.Auth{
			User: conn.Username,
			Pass: conn.Password,
		}
		mechanism := auth.AsSha512Mechanism()

		options = append(options, kgo.SASL(mechanism))
	}

	if conn.TlsEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: conn.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}

		options = append(options, kgo.DialTLSConfig(tlsConfig))
	}

	return options, nil
}
