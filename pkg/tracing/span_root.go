package tracing

import (
	"context"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
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

func newRootSpan(ctx context.Context, name string, identity cfg.Identity, appId string) (context.Context, *awsRootSpan) {
	ctx, cancel := context.WithCancel(ctx)
	ctx, seg := xray.BeginSegment(ctx, name)
	ctx, span := newSpan(ctx, seg, identity, appId)

	transaction := &awsRootSpan{
		span,
		cancel,
	}

	return ctx, transaction
}
