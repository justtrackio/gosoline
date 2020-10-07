package tracing

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-xray-sdk-go/strategy/ctxmissing"
	"github.com/aws/aws-xray-sdk-go/strategy/sampling"
	"github.com/aws/aws-xray-sdk-go/xray"
	"net"
	"net/http"
)

const (
	dnsSrv = "srv"
)

type XRaySettings struct {
	Enabled            bool
	Address            string
	CtxMissingStrategy ctxmissing.Strategy
	SamplingStrategy   sampling.Strategy
}

type awsTracer struct {
	cfg.AppId
	enabled bool
}

func NewAwsTracer(config cfg.Config, logger mon.Logger) Tracer {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	addr := lookupAddr(appId, settings)
	ctxMissingStrategy := NewContextMissingWarningLogStrategy(logger)

	samplingStrategy, err := getSamplingStrategy(&settings.Sampling)
	if err != nil {
		logger.Fatal(err, "could not load sampling strategy, continue with the default")
	}

	xRaySettings := &XRaySettings{
		Enabled:            settings.Enabled,
		Address:            addr,
		CtxMissingStrategy: ctxMissingStrategy,
		SamplingStrategy:   samplingStrategy,
	}

	return NewAwsTracerWithInterfaces(logger, appId, xRaySettings)
}

func NewAwsTracerWithInterfaces(logger mon.Logger, appId cfg.AppId, settings *XRaySettings) *awsTracer {
	err := xray.Configure(xray.Config{
		LogLevel:               "warn",
		DaemonAddr:             settings.Address,
		ContextMissingStrategy: settings.CtxMissingStrategy,
		SamplingStrategy:       settings.SamplingStrategy,
	})

	if err != nil {
		logger.Fatal(err, "can not configure xray tracer")
	}

	return &awsTracer{
		AppId:   appId,
		enabled: settings.Enabled,
	}
}

func (t *awsTracer) StartSubSpan(ctx context.Context, name string) (context.Context, Span) {
	if !t.enabled {
		return ctx, disabledSpan()
	}

	var ctxWithSegment, ctxWithSpan context.Context
	var segment *xray.Segment
	var span Span

	if ctxWithSegment, segment = xray.BeginSubsegment(ctx, name); segment == nil {
		return ctx, disabledSpan()
	}

	ctxWithSpan, span = newSpan(ctxWithSegment, segment, t.AppId)

	return ctxWithSpan, span
}

func (t *awsTracer) StartSpan(name string) (context.Context, Span) {
	if !t.enabled {
		return context.Background(), disabledRootSpan()
	}

	return newRootSpan(context.Background(), name, t.AppId)
}

func (t *awsTracer) StartSpanFromContext(ctx context.Context, name string) (context.Context, Span) {
	if !t.enabled {
		return ctx, disabledSpan()
	}

	parentSpan := GetSpanFromContext(ctx)
	ctx, transaction := newRootSpan(ctx, name, t.AppId)

	if parentSpan != nil {
		parentTrace := parentSpan.GetTrace()
		transaction.awsSpan.segment.TraceID = parentTrace.TraceId
		transaction.awsSpan.segment.ParentID = parentTrace.Id
		transaction.awsSpan.segment.Sampled = parentTrace.Sampled

		return ctx, transaction
	}

	trace := GetTraceFromContext(ctx)

	if trace != nil {
		transaction.awsSpan.segment.TraceID = trace.TraceId
		transaction.awsSpan.segment.ParentID = trace.Id
		transaction.awsSpan.segment.Sampled = trace.Sampled

		return ctx, transaction
	}

	return ctx, transaction
}

func (t *awsTracer) HttpHandler(h http.Handler) http.Handler {
	if !t.enabled {
		return h
	}

	name := fmt.Sprintf("%v-%v-%v-%v", t.Project, t.Environment, t.Family, t.Application)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		seg := xray.GetSegment(r.Context())

		ctx, _ = newSpan(ctx, seg, t.AppId)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})

	return xray.Handler(xray.NewFixedSegmentNamer(name), handlerFunc)
}

func lookupAddr(appId cfg.AppId, settings *TracerSettings) string {
	addressValue := settings.AddressValue

	switch settings.AddressType {
	case dnsSrv:
		if addressValue == "" {
			addressValue = fmt.Sprintf("xray.%v.%v", appId.Environment, appId.Family)
		}

		_, srvs, err := net.LookupSRV("", "", addressValue)

		if err != nil {
			panic(err)
		}

		for _, srv := range srvs {
			addressValue = fmt.Sprintf("%v:%v", srv.Target, srv.Port)
			break
		}
	}

	return addressValue
}

func getSamplingStrategy(samplingConfiguration *SamplingConfiguration) (sampling.Strategy, error) {
	if samplingConfiguration == nil {
		return nil, nil
	}

	samplingConfigurationBytes, err := json.Marshal(samplingConfiguration)
	if err != nil {
		return nil, err
	}

	return sampling.NewLocalizedStrategyFromJSONBytes(samplingConfigurationBytes)
}
