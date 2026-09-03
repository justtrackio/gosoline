package grpcserver

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"google.golang.org/grpc/stats"
)

type key int

const (
	contextKey key = 0

	// MetricRpcServerDuration records how long an RPC took. Its observation count is the request count,
	// so a separate request counter is not needed.
	MetricRpcServerDuration = "duration"

	// The semantic-convention attributes identifying the RPC that was served.
	MetricDimensionRpcService = "rpc.service"
	MetricDimensionRpcMethod  = "rpc.method"
)

type statsHandler struct {
	logger       log.Logger
	metricWriter metric.Writer
	settings     *Settings
}

func NewStatsHandler(logger log.Logger, settings *Settings) *statsHandler {
	writer := metric.NewWriter(metricNamespace)

	return &statsHandler{
		logger:       logger,
		metricWriter: writer,
		settings:     settings,
	}
}

func (s *statsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (s *statsHandler) HandleRPC(ctx context.Context, st stats.RPCStats) {
	holder := ctx.Value(contextKey).(*statsHolder)

	switch v := st.(type) {
	case *stats.Begin:
		holder.BeginTime = v.BeginTime
		holder.FailFast = v.FailFast
		holder.IsClientStream = v.IsClientStream
		holder.IsServerStream = v.IsServerStream
		holder.IsTransparentRetryAttempt = v.IsTransparentRetryAttempt

	case *stats.InHeader:
		holder.InHeaderWireLength = v.WireLength
		holder.InCompression = v.Compression
		holder.FullMethod = v.FullMethod
		if v.RemoteAddr != nil {
			holder.InRemoteAddr = v.RemoteAddr.String()
		}
		if v.LocalAddr != nil {
			holder.InLocalAddr = v.LocalAddr.String()
		}
		for header, value := range v.Header {
			holder.InHeaders.Store(header, value)
		}

	case *stats.InPayload:
		holder.InPayloadLength = v.Length
		holder.InPayloadWireLength = v.WireLength
		holder.RecvTime = v.RecvTime
		if s.settings.Stats.LogPayload {
			holder.InPayload = v.Payload
		}

	case *stats.OutHeader:
		holder.OutCompression = v.Compression
		if v.RemoteAddr != nil {
			holder.OutRemoteAddr = v.RemoteAddr.String()
		}
		if v.LocalAddr != nil {
			holder.OutLocalAddr = v.LocalAddr.String()
		}
		for header, value := range v.Header {
			holder.OutHeaders.Store(header, value)
		}

	case *stats.OutPayload:
		if s.settings.Stats.LogPayload {
			holder.OutPayload = v.Payload
		}
		holder.OutPayloadLength = v.Length
		holder.OutPayloadWireLength = v.WireLength
		holder.SentTime = v.SentTime

	case *stats.End:
		holder.EndTime = v.EndTime
		holder.Error = v.Error
		holder.TotalTime = v.EndTime.Sub(v.BeginTime).Nanoseconds()

		s.writeLog(ctx, holder)
		s.writeMetrics(ctx, holder)
	}
}

func (s *statsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return context.WithValue(ctx, contextKey, &statsHolder{
		InHeaders:  &sync.Map{},
		OutHeaders: &sync.Map{},
	})
}

func (s *statsHandler) HandleConn(_ context.Context, _ stats.ConnStats) {
}

func (s *statsHandler) writeLog(ctx context.Context, holder *statsHolder) {
	logger := s.logger.
		WithFields(holder.GetLoggerFields()).
		WithChannel(s.settings.Stats.Channel)
	msg := "handled gRPC method"

	switch s.settings.Stats.LogLevel {
	case log.LevelDebug:
		logger.Debug(ctx, msg)
	case log.LevelInfo:
		logger.Info(ctx, msg)
	}
}

func (s *statsHandler) writeMetrics(ctx context.Context, holder *statsHolder) {
	service, method := splitFullMethod(holder.FullMethod)

	s.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricRpcServerDuration,
		Dimensions: metric.Dimensions{
			MetricDimensionRpcService: service,
			MetricDimensionRpcMethod:  method,
		},
		Value: float64(holder.TotalTime) / float64(time.Millisecond),
		Unit:  metric.UnitMillisecondsAverage,
		Kind:  metric.KindHistogram.Build(),
	})
}

// splitFullMethod splits a gRPC full method of the form /service/method into the semantic-convention
// service and method attributes. A full method that does not have that shape is reported as the
// service alone, because guessing a method out of it would be worse than reporting none.
func splitFullMethod(fullMethod string) (service string, method string) {
	trimmed := strings.TrimPrefix(fullMethod, "/")

	service, method, found := strings.Cut(trimmed, "/")
	if !found {
		return trimmed, metric.DimensionDefault
	}

	return service, method
}
