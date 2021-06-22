package tracing_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContext(t *testing.T) {
	logger := mocks.NewLoggerMockedAll()

	tracer, err := tracing.NewAwsTracerWithInterfaces(logger, cfg.AppId{}, &tracing.XRaySettings{Enabled: true})
	assert.NoError(t, err, "we should be able to get a tracer")

	ctx, span := tracer.StartSpan("test")

	spanFromCtx := tracing.GetSpanFromContext(ctx)

	assert.NotEmpty(t, span.GetTrace().Id)
	assert.Equal(t, span.GetTrace().Id, spanFromCtx.GetTrace().Id)
}
