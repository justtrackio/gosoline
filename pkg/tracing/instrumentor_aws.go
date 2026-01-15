package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func init() {
	AddInstrumentorProvider(ProviderXray, NewAwsInstrumentor)
}

type awsInstrumentor struct {
	cfg.Identity
	appId string
}

func NewAwsInstrumentor(_ context.Context, config cfg.Config, _ log.Logger) (Instrumentor, error) {
	identity, err := cfg.GetAppIdentity(config)
	if err != nil {
		return nil, fmt.Errorf("could not get app identity from config: %w", err)
	}

	appId, err := resolveAppId(config)
	if err != nil {
		return nil, fmt.Errorf("failed to format service appId: %w", err)
	}

	return NewAwsInstrumentorWithInterfaces(identity, appId), nil
}

func NewAwsInstrumentorWithInterfaces(identity cfg.Identity, appId string) *awsInstrumentor {
	return &awsInstrumentor{
		Identity: identity,
		appId:    appId,
	}
}

func (t *awsInstrumentor) HttpHandler(h http.Handler) http.Handler {
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		seg := xray.GetSegment(ctx)

		ctx, _ = newSpan(ctx, seg, t.Identity, t.appId)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})

	return xray.Handler(xray.NewFixedSegmentNamer(t.appId), handlerFunc)
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
