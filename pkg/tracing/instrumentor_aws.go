package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func init() {
	AddInstrumentorProvider(ProviderXray, NewAwsInstrumentor)
}

type awsInstrumentor struct {
	cfg.AppIdentity
}

func NewAwsInstrumentor(_ context.Context, config cfg.Config, _ log.Logger) (Instrumentor, error) {
	identity, err := cfg.GetAppIdentityFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not get app identity from config: %w", err)
	}

	return NewAwsInstrumentorWithInterfaces(identity), nil
}

func NewAwsInstrumentorWithInterfaces(identity cfg.AppIdentity) *awsInstrumentor {
	return &awsInstrumentor{
		AppIdentity: identity,
	}
}

func (t *awsInstrumentor) HttpHandler(h http.Handler) http.Handler {
	name := fmt.Sprintf("%s-%s-%s-%s-%s", t.Tags.Get("project"), t.Env, t.Tags.Get("family"), t.Tags.Get("group"), t.Name)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		seg := xray.GetSegment(ctx)

		ctx, _ = newSpan(ctx, seg, t.AppIdentity)
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

func (t *awsInstrumentor) GrpcServerHandler() stats.Handler {
	return nil
}
