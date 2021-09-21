package cast_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cast"
	"github.com/justtrackio/gosoline/pkg/test/assert"
)

func TestToSlicePtrString(t *testing.T) {
	in := []string{"1", "2", "3"}
	out := cast.ToSlicePtrString(in)
	for i, v := range in {
		assert.Equal(t, v, *out[i])
	}
}
