package tracing

import (
	"context"
	"fmt"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
)

//go:generate mockery --name Span
type Span interface {
	AddAnnotation(key string, value string)
	AddError(err error)
	AddMetadata(key string, value interface{})
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

	_ = s.segment.AddAnnotation(key, value)
}

func (s awsSpan) AddError(err error) {
	if !s.enabled {
		return
	}

	_ = s.segment.AddError(err)
}

func (s awsSpan) AddMetadata(key string, value interface{}) {
	if !s.enabled {
		return
	}

	_ = s.segment.AddMetadata(key, value)
}

func (s awsSpan) Finish() {
	if !s.enabled {
		return
	}

	s.segment.Close(nil)
}

func newSpan(ctx context.Context, seg *xray.Segment, app cfg.AppId) (context.Context, *awsSpan) {
	span := &awsSpan{
		enabled: true,
		segment: seg,
	}

	appFamily := fmt.Sprintf("%s-%s-%s", app.Project, app.Environment, app.Family)
	appId := fmt.Sprintf("%s-%s-%s-%s-%s", app.Project, app.Environment, app.Family, app.Group, app.Application)
	span.AddAnnotation("appFamily", appFamily)
	span.AddAnnotation("appId", appId)

	return ContextWithSpan(ctx, span), span
}

func disabledSpan() *awsSpan {
	return &awsSpan{
		enabled: false,
	}
}
