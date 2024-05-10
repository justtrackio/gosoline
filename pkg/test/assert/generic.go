package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Equal(t *testing.T, expected, actual any, msgAndArgs ...any) {
	assert.Equal(t, expected, actual, msgAndArgs...)
}
