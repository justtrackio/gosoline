package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	assert.Equal(t, expected, actual, msgAndArgs...)
}
