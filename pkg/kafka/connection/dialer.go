package connection

import (
	"crypto/tls"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/scram"
)

const (
	// DefaultDialerTimeout is how long to wait for TCP connection to be established with bootstrap.
	DefaultDialerTimeout = 10 * time.Second

	// DefaultKeepAlive is how long an unused connection should be kept open in the hope of re-use.
	DefaultKeepAlive = 10 * time.Minute
)

// NewDialer is a dialer factory.
func NewDialer(conf *Settings) (*kafka.Dialer, error) {
	var err error
	var mechanism sasl.Mechanism
	var tlsConfig *tls.Config

	if conf.Username != "" && conf.Password != "" {
		mechanism, err = scram.Mechanism(scram.SHA512, conf.Username, conf.Password)
		if err != nil {
			return nil, err
		}
	}

	if conf.TlsEnabled {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: conf.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
	}

	return &kafka.Dialer{
		DualStack:       true,
		TLS:             tlsConfig,
		SASLMechanism:   mechanism,
		KeepAlive:       DefaultKeepAlive,
		Timeout:         DefaultDialerTimeout,
		TransactionalID: uuid.New().String(),
	}, nil
}
