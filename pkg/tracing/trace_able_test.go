package tracing_test

import (
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/twinj/uuid"
	"testing"
)

func TestTrace(t *testing.T) {
	id := uuid.NewV4().String()
	parentId := uuid.NewV4().String()

	trace := tracing.Trace{
		Id:       id,
		ParentId: parentId,
		Sampled:  true,
	}

	assert.Equal(t, id, trace.GetId())
	assert.Equal(t, parentId, trace.GetParentId())
	assert.Equal(t, true, trace.GetSampled())
}
