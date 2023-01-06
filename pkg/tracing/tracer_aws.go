package tracing

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/strategy/ctxmissing"
	"github.com/aws/aws-xray-sdk-go/strategy/sampling"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	dnsSrv                        = "srv"
	xrayDefaultMaxSubsegmentCount = 20
)

type XRaySettings struct {
	Enabled                     bool
	Address                     string
	CtxMissingStrategy          ctxmissing.Strategy
	SamplingStrategy            sampling.Strategy
	StreamingMaxSubsegmentCount int
}

type awsTracer struct {
	cfg.AppId
	enabled bool
}

func NewAwsTracer(config cfg.Config, logger log.Logger) (Tracer, error) {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	addr := lookupAddr(appId, settings)
	ctxMissingStrategy := NewContextMissingWarningLogStrategy(logger)

	samplingStrategy, err := getSamplingStrategy(&settings.Sampling)
	if err != nil {
		return nil, fmt.Errorf("could not load sampling strategy: %w", err)
	}

	xRaySettings := &XRaySettings{
		Enabled:                     settings.Enabled,
		Address:                     addr,
		CtxMissingStrategy:          ctxMissingStrategy,
		SamplingStrategy:            samplingStrategy,
		StreamingMaxSubsegmentCount: settings.StreamingMaxSubsegmentCount,
	}

	return NewAwsTracerWithInterfaces(logger, appId, xRaySettings)
}

func NewAwsTracerWithInterfaces(logger log.Logger, appId cfg.AppId, settings *XRaySettings) (*awsTracer, error) {
	if settings.StreamingMaxSubsegmentCount == 0 {
		settings.StreamingMaxSubsegmentCount = xrayDefaultMaxSubsegmentCount
	}

	streamingStrategy, err := xray.NewDefaultStreamingStrategyWithMaxSubsegmentCount(settings.StreamingMaxSubsegmentCount)
	if err != nil {
		return nil, fmt.Errorf("can not create default xray streaming strategy: %w", err)
	}

	err = xray.Configure(xray.Config{
		LogLevel:               "warn",
		DaemonAddr:             settings.Address,
		ContextMissingStrategy: settings.CtxMissingStrategy,
		SamplingStrategy:       settings.SamplingStrategy,
		StreamingStrategy:      streamingStrategy,
	})

	if err != nil {
		return nil, fmt.Errorf("can not configure xray tracer: %w", err)
	}

	xrayLogger := newXrayLogger(logger)
	xray.SetLogger(xrayLogger)

	return &awsTracer{
		AppId:   appId,
		enabled: settings.Enabled,
	}, nil
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
		transaction.awsSpan.segment.ParentID = trace.ParentId
		transaction.awsSpan.segment.Sampled = trace.Sampled

		return ctx, transaction
	}

	return ctx, transaction
}

func (t *awsTracer) HttpHandler(h http.Handler) http.Handler {
	if !t.enabled {
		return h
	}

	name := fmt.Sprintf("%s-%s-%s-%s-%s", t.Project, t.Environment, t.Family, t.Group, t.Application)
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
