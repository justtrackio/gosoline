package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc"
)

type awsInstrumentor struct {
	cfg.AppId
}

func NewAwsInstrumentor(_ context.Context, config cfg.Config, _ log.Logger) (Instrumentor, error) {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	return NewAwsInstrumentorWithAppId(appId), nil
}

func NewAwsInstrumentorWithAppId(appId cfg.AppId) *awsInstrumentor {
	return &awsInstrumentor{
		AppId: appId,
	}
}

func (t *awsInstrumentor) HttpHandler(h http.Handler) http.Handler {
	name := fmt.Sprintf("%s-%s-%s-%s-%s", t.Project, t.Environment, t.Family, t.Group, t.Application)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		seg := xray.GetSegment(ctx)

		ctx, _ = newSpan(ctx, seg, t.AppId)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})

	return xray.Handler(xray.NewFixedSegmentNamer(name), handlerFunc)
}

func (t *awsInstrumentor) HttpClient(baseClient *http.Client) *http.Client {
	return xray.Client(baseClient)
}

func (t *awsInstrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return xray.UnaryServerInterceptor()
}
