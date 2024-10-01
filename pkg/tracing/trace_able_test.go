package tracing_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTrace(t *testing.T) {
	id := uuid.New().NewV4()
	parentId := uuid.New().NewV4()

	trace := tracing.Trace{
		Id:       id,
		ParentId: parentId,
		Sampled:  true,
	}

	assert.Equal(t, id, trace.GetId())
	assert.Equal(t, parentId, trace.GetParentId())
	assert.Equal(t, true, trace.GetSampled())
}

func TestTraceToString(t *testing.T) {
	trace := &tracing.Trace{
		TraceId:  "1-5759e988-bd862e3fe1be46a994272793",
		Id:       "54567a67e89cdf88",
		ParentId: "53995c3f42cd8ad8",
		Sampled:  true,
	}

	traceId := tracing.TraceToString(trace)

	assert.Equal(t, "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=54567a67e89cdf88;Sampled=1", traceId)
}

func TestStringToTrace(t *testing.T) {
	_, err := tracing.StringToTrace("Root=1-5759e988-bd862e3fe1be46a994272793")
	assert.EqualErrorf(t, err, "the trace id [Root=1-5759e988-bd862e3fe1be46a994272793] should consist of at least 2 parts and at most of 3 parts", "error does not match")

	_, err = tracing.StringToTrace("Root=1-5759e988-bd862e3fe1be46a994272793;Parent")
	assert.EqualErrorf(t, err, "the sampled part [Parent] of the trace id seems malformed", "error does not match")

	trace, err := tracing.StringToTrace("Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=0")
	assert.NoError(t, err)
	assert.Equal(t, "1-5759e988-bd862e3fe1be46a994272793", trace.GetTraceId())
	assert.Equal(t, "", trace.GetParentId())
	assert.Equal(t, false, trace.GetSampled())

	trace, err = tracing.StringToTrace("Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")
	assert.NoError(t, err)
	assert.Equal(t, "1-5759e988-bd862e3fe1be46a994272793", trace.GetTraceId())
	assert.Equal(t, "53995c3f42cd8ad8", trace.GetParentId())
	assert.Equal(t, true, trace.GetSampled())
}
