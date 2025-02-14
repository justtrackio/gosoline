package grpcserver

import (
	"context"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"google.golang.org/grpc/stats"
)

type key int

const (
	contextKey key = 0

	MetricApiRequestCount        = "ApiRequestCount"
	MetricApiRequestResponseTime = "ApiRequestResponseTime"
	MetricDimensionFullMethod    = "full_method"
)

type statsHandler struct {
	logger       log.Logger
	metricWriter metric.Writer
	settings     *Settings
}

func NewStatsHandler(ctx context.Context, logger log.Logger, settings *Settings) *statsHandler {
	writer := metric.NewWriter(ctx)

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
		s.writeMetrics(holder)
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
		WithContext(ctx).
		WithFields(holder.GetLoggerFields()).
		WithChannel(s.settings.Stats.Channel)
	msg := "handled gRPC method"

	switch s.settings.Stats.LogLevel {
	case log.LevelDebug:
		logger.Debug(msg)
	case log.LevelInfo:
		logger.Info(msg)
	}
}

func (s *statsHandler) writeMetrics(holder *statsHolder) {
	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricApiRequestResponseTime,
		Dimensions: metric.Dimensions{
			MetricDimensionFullMethod: holder.FullMethod,
		},
		Value: float64(holder.TotalTime) / float64(time.Millisecond),
		Unit:  metric.UnitMillisecondsAverage,
	})

	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricApiRequestCount,
		Dimensions: metric.Dimensions{
			MetricDimensionFullMethod: holder.FullMethod,
		},
		Value: 1.0,
		Unit:  metric.UnitCount,
	})
}
