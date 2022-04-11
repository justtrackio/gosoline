package mdl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBool(t *testing.T) {
	input := true
	got := Bool(input)

	assert.Equal(t, input, *got)
}


func FuzzBool(f *testing.F) {
	f.Add(true)
	f.Add(false)

	f.Fuzz(func(t *testing.T, input bool){
		got := Bool(input)

		assert.Equal(t, input, *got)
	})
}
