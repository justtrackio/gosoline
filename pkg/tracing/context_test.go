package tracing_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContext(t *testing.T) {
	tracer := tracing.NewAwsTracerWithInterfaces(cfg.AppId{}, tracing.Settings{Enabled: true})
	ctx, span := tracer.StartSpan("test")

	spanFromCtx := tracing.GetSpan(ctx)

	assert.NotEmpty(t, span.GetTrace().Id)
	assert.Equal(t, span.GetTrace().Id, spanFromCtx.GetTrace().Id)
}
