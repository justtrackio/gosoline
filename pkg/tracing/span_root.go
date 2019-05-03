package tracing

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type awsRootSpan struct {
	*awsSpan
	cancel context.CancelFunc
}

func (t awsRootSpan) Finish() {
	if !t.enabled {
		return
	}

	t.awsSpan.Finish()

	if t.cancel == nil {
		return
	}

	t.cancel()
}

func newRootSpan(ctx context.Context, name string, app cfg.AppId) (context.Context, *awsRootSpan) {
	ctx, cancel := context.WithCancel(ctx)
	ctx, seg := xray.BeginSegment(ctx, name)
	ctx, span := newSpan(ctx, seg, app)

	transaction := &awsRootSpan{
		span,
		cancel,
	}

	return ctx, transaction
}

func disabledRootSpan() *awsRootSpan {
	return &awsRootSpan{
		&awsSpan{
			enabled: false,
		},
		nil,
	}
}
