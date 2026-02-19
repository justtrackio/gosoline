package tracing

import (
	"context"
	"fmt"
	"net"

	"github.com/aws/aws-xray-sdk-go/v2/strategy/ctxmissing"
	"github.com/aws/aws-xray-sdk-go/v2/strategy/sampling"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	AddTracerProvider(ProviderXray, NewAwsTracer)
}

const (
	dnsSrv                        = "srv"
	xrayDefaultMaxSubsegmentCount = 20
)

type XrayTracerSettings struct {
	AddressType                 string                      `cfg:"addr_type" default:"local" validate:"required"`
	AddressValue                string                      `cfg:"add_value" default:""`
	Naming                      XrayTracerSrvNamingSettings `cfg:"srv_naming"`
	Sampling                    SamplingConfiguration       `cfg:"sampling"`
	StreamingMaxSubsegmentCount int                         `cfg:"streaming_max_subsegment_count" default:"20"`
}

type XrayTracerSrvNamingSettings struct {
	Pattern   string `cfg:"pattern,nodecode" default:"xray.{app.namespace}"`
	Delimiter string `cfg:"delimiter" default:"."`
}

type XRaySettings struct {
	Address                     string
	CtxMissingStrategy          ctxmissing.Strategy
	SamplingStrategy            sampling.Strategy
	StreamingMaxSubsegmentCount int
}

type awsTracer struct {
	cfg.Identity
	appId string
}

func NewAwsTracer(_ context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	identity, err := cfg.GetAppIdentity(config)
	if err != nil {
		return nil, fmt.Errorf("could not get app identity from config: %w", err)
	}

	settings := &XrayTracerSettings{}
	if err := config.UnmarshalKey("tracing.xray", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal xray tracer settings: %w", err)
	}

	appId, err := resolveAppId(config)
	if err != nil {
		return nil, fmt.Errorf("failed to format app id: %w", err)
	}

	addr, err := lookupAddr(config, identity, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup address: %w", err)
	}
	ctxMissingStrategy := NewContextMissingWarningLogStrategy(logger)

	samplingStrategy, err := getSamplingStrategy(&settings.Sampling)
	if err != nil {
		return nil, fmt.Errorf("could not load sampling strategy: %w", err)
	}

	xRaySettings := &XRaySettings{
		Address:                     addr,
		CtxMissingStrategy:          ctxMissingStrategy,
		SamplingStrategy:            samplingStrategy,
		StreamingMaxSubsegmentCount: settings.StreamingMaxSubsegmentCount,
	}

	return NewAwsTracerWithInterfaces(logger, identity, xRaySettings, appId)
}

func NewAwsTracerWithInterfaces(logger log.Logger, identity cfg.Identity, settings *XRaySettings, appId string) (*awsTracer, error) {
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

	setGlobalXRayLogger(logger)

	return &awsTracer{
		Identity: identity,
		appId:    appId,
	}, nil
}

func (t *awsTracer) StartSubSpan(ctx context.Context, name string) (context.Context, Span) {
	var ctxWithSegment context.Context
	var segment *xray.Segment

	if ctxWithSegment, segment = xray.BeginSubsegment(ctx, name); segment == nil {
		return ctx, disabledSpan()
	}

	return newSpan(ctxWithSegment, segment, t.Identity, t.appId)
}

func (t *awsTracer) StartSpan(name string) (context.Context, Span) {
	return newRootSpan(context.Background(), name, t.Identity, t.appId)
}

func (t *awsTracer) StartSpanFromContext(ctx context.Context, name string) (context.Context, Span) {
	parentSpan := GetSpanFromContext(ctx)
	ctx, transaction := newRootSpan(ctx, name, t.Identity, t.appId)

	if parentSpan != nil {
		parentTrace := parentSpan.GetTrace()
		transaction.segment.TraceID = parentTrace.TraceId
		transaction.segment.ParentID = parentTrace.Id
		transaction.segment.Sampled = parentTrace.Sampled

		return ctx, transaction
	}

	trace := GetTraceFromContext(ctx)

	if trace != nil {
		transaction.segment.TraceID = trace.TraceId
		transaction.segment.ParentID = trace.ParentId
		transaction.segment.Sampled = trace.Sampled

		return ctx, transaction
	}

	return ctx, transaction
}

func lookupAddr(config cfg.Config, identity cfg.Identity, settings *XrayTracerSettings) (string, error) {
	addressValue := settings.AddressValue

	if settings.AddressType != dnsSrv {
		return addressValue, nil
	}

	var err error
	var srvName string
	var srvs []*net.SRV

	if addressValue == "" {
		if srvName, err = identity.Format(settings.Naming.Pattern, settings.Naming.Delimiter); err != nil {
			return "", fmt.Errorf("failed to format srv name: %w", err)
		}

		addressValue = srvName
	}

	if _, srvs, err = net.LookupSRV("", "", addressValue); err != nil {
		return "", fmt.Errorf("failed to lookup srv records for %s: %w", addressValue, err)
	}

	for _, srv := range srvs {
		addressValue = fmt.Sprintf("%v:%v", srv.Target, srv.Port)

		break
	}

	return addressValue, nil
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
