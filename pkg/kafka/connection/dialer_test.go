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
		})

	assert.Nil(t, err)
	assert.True(t, dialer.TLS.InsecureSkipVerify)
	assert.NotNil(t, dialer.SASLMechanism)
	assert.NotEmpty(t, dialer.TransactionalID)
}
