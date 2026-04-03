package mdl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
)

func TestClonePtr(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, mdl.ClonePtr[string](nil))
	})

	t.Run("clones value", func(t *testing.T) {
		value := "hello"

		cloned := mdl.ClonePtr(&value)

		if assert.NotNil(t, cloned) {
			assert.Equal(t, value, *cloned)
			assert.NotSame(t, &value, cloned)
		}

		*cloned = "changed"

		assert.Equal(t, "hello", value)
	})
}
