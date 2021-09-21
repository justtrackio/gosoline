package coffin_test

import (
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestResolveRecovery(t *testing.T) {
	err := errors.New("special error")

	assert.Nil(t, coffin.ResolveRecovery(nil))
	assert.Error(t, coffin.ResolveRecovery("string error"))
	assert.Error(t, coffin.ResolveRecovery(err))
	assert.Error(t, coffin.ResolveRecovery(42))

	assert.True(t, strings.Contains(coffin.ResolveRecovery("string error").Error(), "string error"))
	assert.True(t, errors.Is(coffin.ResolveRecovery(err), err))
	assert.True(t, strings.Contains(coffin.ResolveRecovery(42).Error(), "unhandled error type int"))
}
