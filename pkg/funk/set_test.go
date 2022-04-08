package funk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet_Set(t *testing.T) {
	s := Set[string]{}

	s.Set("a")
	s.Set("b")

	assert.Contains(t, s, "a")
	assert.Contains(t, s, "b")
}

func TestSet_Contains(t *testing.T) {
	s := Set[int]{}

	assert.NotContains(t, s, 123)

	s.Set(123)
	assert.Contains(t, s, 123)
}

func FuzzSet(f *testing.F) {
	f.Add([]byte("my byte slice"))

	f.Fuzz(func(t *testing.T, input []byte) {
		set := Set[byte]{}

		for _, v := range input {
			set.Set(v)
		}

		for _, v := range input {
			assert.Contains(t, set, v)
		}
	})
}
