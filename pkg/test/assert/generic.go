package assert

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Equal(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	assert.Equal(t, expected, actual, msgAndArgs...)
}
