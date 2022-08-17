package mdlsub_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/stretchr/testify/assert"
)

func TestDelayOpError(t *testing.T) {
	err := mdlsub.NewDelayOpError(fmt.Errorf("fail"))
	assert.True(t, mdlsub.IsDelayOpError(err))
	assert.False(t, mdlsub.IsDelayOpError(err.Unwrap()))

	assert.EqualError(t, err, "delayed op error: fail")
}
