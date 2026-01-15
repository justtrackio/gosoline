package tracing

import (
	"context"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

//go:generate go run github.com/vektra/mockery/v2 --name Span
type Span interface {
	AddAnnotation(key string, value string)
	AddError(err error)
	AddMetadata(key string, value any)
	Finish()
	GetId() string
	GetTrace() *Trace
}

type awsSpan struct {
	enabled bool
	segment *xray.Segment
}

func (s awsSpan) GetId() string {
	return s.segment.ID
}

func (s awsSpan) GetTrace() *Trace {
	if !s.enabled {
		return &Trace{}
	}

	seg := s.segment
	for seg.ParentSegment != seg {
		seg = seg.ParentSegment
	}

	return &Trace{
		TraceId:  seg.TraceID,
		Id:       s.segment.ID,
		ParentId: seg.ParentID,
		Sampled:  seg.Sampled,
	}
}

func (s awsSpan) AddAnnotation(key string, value string) {
	if !s.enabled {
		return
	}

	_ = s.segment.AddAnnotation(key, value) //nolint:errcheck // best-effort tracing annotation
}

func (s awsSpan) AddError(err error) {
	if !s.enabled {
		return
	}

	_ = s.segment.AddError(err) //nolint:errcheck // best-effort tracing error recording
}

func (s awsSpan) AddMetadata(key string, value any) {
	if !s.enabled {
		return
	}

	_ = s.segment.AddMetadata(key, value) //nolint:errcheck // best-effort tracing metadata
}

func (s awsSpan) Finish() {
	if !s.enabled {
		return
	}

	s.segment.Close(nil)
}

func newSpan(ctx context.Context, seg *xray.Segment, identity cfg.Identity, appId string) (context.Context, *awsSpan) {
	span := &awsSpan{
		enabled: true,
		segment: seg,
	}

	span.AddAnnotation("appId", appId)

	return ContextWithSpan(ctx, span), span
}

func disabledSpan() *awsSpan {
	return &awsSpan{
		enabled: false,
	}
}
