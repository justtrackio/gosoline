package tracing_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContext(t *testing.T) {
	logger := mocks.NewLoggerMockedAll()

	tracer := tracing.NewAwsTracerWithInterfaces(logger, cfg.AppId{}, &tracing.XRaySettings{Enabled: true})
	ctx, span := tracer.StartSpan("test")

	spanFromCtx := tracing.GetSpanFromContext(ctx)

	assert.NotEmpty(t, span.GetTrace().Id)
	assert.Equal(t, span.GetTrace().Id, spanFromCtx.GetTrace().Id)
}
