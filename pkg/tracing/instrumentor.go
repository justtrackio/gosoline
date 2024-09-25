package tracing

import (
	"net/http"

	"google.golang.org/grpc"
)

//go:generate mockery --name Instrumentor
type Instrumentor interface {
	HttpHandler(h http.Handler) http.Handler
	HttpClient(baseClient *http.Client) *http.Client
	GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor
}
