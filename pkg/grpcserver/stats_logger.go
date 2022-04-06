package grpcserver

import (
	"context"
	"sync"

	"google.golang.org/grpc/stats"

	"github.com/justtrackio/gosoline/pkg/log"
)

type key int

const (
	contextKey key = 0
)

type statsLogger struct {
	logger   log.Logger
	settings *Settings
}

func NewStatsLogger(logger log.Logger, settings *Settings) *statsLogger {
	return &statsLogger{
		logger:   logger,
		settings: settings,
	}
}

func (s *statsLogger) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (s *statsLogger) HandleRPC(ctx context.Context, st stats.RPCStats) {
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
		if s.settings.Stats.LogData {
			holder.InData = v.Data
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
		if s.settings.Stats.LogData {
			holder.OutData = v.Data
		}
		holder.OutPayloadLength = v.Length
		holder.OutPayloadWireLength = v.WireLength
		holder.SentTime = v.SentTime

	case *stats.End:
		holder.EndTime = v.EndTime
		holder.Error = v.Error
		holder.TotalTime = v.EndTime.Sub(v.BeginTime).Nanoseconds()

		s.logger.
			WithContext(ctx).
			WithFields(holder.GetLoggerFields()).
			WithChannel(s.settings.Stats.Channel).
			Debug("handled gRPC method")
	}
}

func (s *statsLogger) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return context.WithValue(ctx, contextKey, &statsHolder{
		InHeaders:  &sync.Map{},
		OutHeaders: &sync.Map{},
	})
}

func (s *statsLogger) HandleConn(_ context.Context, _ stats.ConnStats) {
}
