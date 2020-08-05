package cast_test

import (
	"github.com/applike/gosoline/pkg/cast"
	"github.com/applike/gosoline/pkg/test/assert"
	"testing"
)

func TestToSlicePtrString(t *testing.T) {
	in := []string{"1", "2", "3"}
	out := cast.ToSlicePtrString(in)
	for i, v := range in {
		assert.Equal(t, v, *out[i])
	}
}
