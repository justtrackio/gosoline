package fixtures_test

import (
	"encoding/hex"
	"testing"

	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestByteSequence_DifferentSeed(t *testing.T) {
	byteSequence1 := fixtures.NewByteSequence("test")
	byteSequence2 := fixtures.NewByteSequence("something else")

	for i := 0; i < 100; i++ {
		assert.NotEqual(t, byteSequence1.GetBytes(32), byteSequence2.GetBytes(32))
	}
}

func TestByteSequence_SameSeed(t *testing.T) {
	byteSequence1 := fixtures.NewByteSequence("test")
	byteSequence2 := fixtures.NewByteSequence("test")

	for i := 0; i < 100; i++ {
		assert.Equal(t, byteSequence1.GetBytes(32), byteSequence2.GetBytes(32))
	}
}

func TestByteSequence_RandomValues(t *testing.T) {
	byteSequence := fixtures.NewByteSequence("test")
	seenValues := map[string]bool{}

	for i := 0; i < 100; i++ {
		encoded := hex.EncodeToString(byteSequence.GetBytes(32))

		_, ok := seenValues[encoded]
		assert.False(t, ok)

		seenValues[encoded] = true
	}
}
