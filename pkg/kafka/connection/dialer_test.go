package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	username = "username"
	password = "password"
)

func TestNewDialer(t *testing.T) {
	dialer, err := NewDialer(
		&Settings{
			Username:           username,
			Password:           password,
			InsecureSkipVerify: true,
			TlsEnabled:         true,
		})

	assert.Nil(t, err)
	assert.True(t, dialer.TLS.InsecureSkipVerify)
	assert.NotNil(t, dialer.SASLMechanism)
	assert.NotEmpty(t, dialer.TransactionalID)
}

func TestNewDialer_NoAuthAndNoTLS(t *testing.T) {
	dialer, err := NewDialer(
		&Settings{
			TlsEnabled: false,
		})

	assert.Nil(t, err)
	assert.Nil(t, dialer.TLS)
	assert.Nil(t, dialer.SASLMechanism)
	assert.NotEmpty(t, dialer.TransactionalID)
}
