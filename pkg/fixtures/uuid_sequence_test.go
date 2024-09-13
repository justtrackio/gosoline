package fixtures_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUuidSequence_DifferentSeed(t *testing.T) {
	uuidSequence1 := fixtures.NewUuidSequence("test")
	uuidSequence2 := fixtures.NewUuidSequence("something else")

	for i := 0; i < 100; i++ {
		assert.NotEqual(t, uuidSequence1.NewV4(), uuidSequence2.NewV4())
	}
}

func TestUuidSequence_SameSeed(t *testing.T) {
	uuidSequence1 := fixtures.NewUuidSequence("test")
	uuidSequence2 := fixtures.NewUuidSequence("test")

	for i := 0; i < 100; i++ {
		assert.Equal(t, uuidSequence1.NewV4(), uuidSequence2.NewV4())
	}
}

func TestUuidSequence_ValidResults(t *testing.T) {
	uuidSequence := fixtures.NewUuidSequence("test")

	for i := 0; i < 100; i++ {
		assert.True(t, uuid.ValidV4(uuidSequence.NewV4()))
	}
}
