package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type bla struct{}

func (b bla) GetId() any {
	return 1
}

func TestGetValueId(t *testing.T) {
	foo := bla{}
	id, _ := GetValueId(foo)

	assert.Equal(t, id, 1)
}
