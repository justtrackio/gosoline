package tracing

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-xray-sdk-go/xray"
	"net"
	"net/http"
)

const (
	dnsSrv = "srv"
)

type Settings struct {
	Enabled bool
	Addr    string
}

type awsTracer struct {
	cfg.AppId
	enabled bool
}

func NewAwsTracer(config cfg.Config) *awsTracer {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	addr := lookupAddr(appId, config)

	settings := Settings{
		Enabled: config.GetBool("tracing_enabled"),
		Addr:    addr,
	}

	return NewAwsTracerWithInterfaces(appId, settings)
}

func NewAwsTracerWithInterfaces(appId cfg.AppId, settings Settings) *awsTracer {
	err := xray.Configure(xray.Config{
		LogLevel:   "warn",
		DaemonAddr: settings.Addr,
	})

	if err != nil {
		panic(err)
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

	ctx, seg := xray.BeginSubsegment(ctx, name)
	ctx, span := newSpan(ctx, seg, t.AppId)

	return ctx, span
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

	parentSpan := GetSpan(ctx)
	ctx, transaction := newRootSpan(ctx, name, t.AppId)

	if parentSpan == nil {
		return ctx, transaction
	}

	parentTrace := parentSpan.GetTrace()
	transaction.awsSpan.segment.TraceID = parentTrace.TraceId
	transaction.awsSpan.segment.ParentID = parentTrace.Id
	transaction.awsSpan.segment.Sampled = parentTrace.Sampled

	return ctx, transaction
}

func (t *awsTracer) StartSpanFromTraceAble(obj TraceAble, name string) (context.Context, Span) {
	if !t.enabled {
		return context.Background(), disabledSpan()
	}

	ctx, transaction := newRootSpan(context.Background(), name, t.AppId)

	trace := obj.GetTrace()
	if trace != nil {
		transaction.awsSpan.segment.TraceID = trace.GetTraceId()
		transaction.awsSpan.segment.ParentID = trace.GetId()
		transaction.awsSpan.segment.Sampled = trace.GetSampled()
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

func lookupAddr(appId cfg.AppId, config cfg.Config) string {
	addrType := config.GetString("tracing_addr_type")
	addrValue := config.GetString("tracing_addr_value")

	switch addrType {
	case dnsSrv:
		if addrValue == "" {
			addrValue = fmt.Sprintf("xray.%v.%v", appId.Environment, appId.Family)
		}

		_, srvs, err := net.LookupSRV("", "", addrValue)

		if err != nil {
			panic(err)
		}

		for _, srv := range srvs {
			addrValue = fmt.Sprintf("%v:%v", srv.Target, srv.Port)
			break
		}
	}

	return addrValue
}
