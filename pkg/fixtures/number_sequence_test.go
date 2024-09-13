package fixtures_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
)

func TestNumberSequence_NextInt(t *testing.T) {
	autoNumbered := fixtures.NewNumberSequence()
	assert.Equal(t, 1, autoNumbered.GetNextInt())
	assert.Equal(t, 2, autoNumbered.GetNextInt())
	assert.Equal(t, 3, autoNumbered.GetNextInt())
}

func TestNumberSequence_NextId(t *testing.T) {
	autoNumbered := fixtures.NewNumberSequence()
	assert.Equal(t, mdl.Box[uint](1), autoNumbered.GetNextId())
	assert.Equal(t, mdl.Box[uint](2), autoNumbered.GetNextId())
	assert.Equal(t, mdl.Box[uint](3), autoNumbered.GetNextId())
}

func TestNumberSequenceFrom(t *testing.T) {
	autoNumbered := fixtures.NewNumberSequenceFrom(5)
	assert.Equal(t, uint(5), autoNumbered.GetNextUint())
	assert.Equal(t, uint(6), autoNumbered.GetNextUint())
	assert.Equal(t, uint(7), autoNumbered.GetNextUint())
}
