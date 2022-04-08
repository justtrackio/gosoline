package funk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet_Set(t *testing.T) {
	s := Set[string]{}

	s.Set("a")
	s.Set("b")

	assert.True(t, s.Contains("a"))
	assert.True(t, s.Contains("b"))
}

func TestSet_Contains(t *testing.T) {
	s := Set[int]{}

	assert.False(t, s.Contains(123))

	s.Set(123)
	assert.True(t, s.Contains(123))
}
