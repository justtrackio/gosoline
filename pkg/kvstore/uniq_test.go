package kvstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniqKeys_StringKeys(t *testing.T) {
	keys := []interface{}{"a", "b", "b"}

	uniqKeys, err := UniqKeys(keys)

	assert.NoError(t, err)
	assert.Len(t, uniqKeys, 2)
	assert.Contains(t, uniqKeys, "a")
	assert.Contains(t, uniqKeys, "b")
}

func TestUniqKeys_IntegerKeys(t *testing.T) {
	keys := []interface{}{1, 2, 1}

	uniqKeys, err := UniqKeys(keys)

	assert.NoError(t, err)
	assert.Len(t, uniqKeys, 2)
	assert.Contains(t, uniqKeys, 1)
	assert.Contains(t, uniqKeys, 2)
}

func TestUniqKeys_MixedKeys(t *testing.T) {
	keys := []interface{}{1, "2", "1"}

	uniqKeys, err := UniqKeys(keys)

	assert.NoError(t, err)
	assert.Len(t, uniqKeys, 2)
	assert.Contains(t, uniqKeys, 1)
	assert.Contains(t, uniqKeys, "2")
}
