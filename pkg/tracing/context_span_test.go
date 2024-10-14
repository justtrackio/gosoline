package tracing_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	tracer, err := tracing.NewAwsTracerWithInterfaces(logger, cfg.AppId{}, &tracing.XRaySettings{})
	assert.NoError(t, err, "we should be able to get a tracer")

	ctx, span := tracer.StartSpan("test")

	spanFromCtx := tracing.GetSpanFromContext(ctx)

	assert.NotEmpty(t, span.GetTrace().Id)
	assert.Equal(t, span.GetTrace().Id, spanFromCtx.GetTrace().Id)
}
