package tracing

import (
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

//go:generate go run github.com/vektra/mockery/v2 --name Instrumentor
type Instrumentor interface {
	HttpHandler(h http.Handler) http.Handler
	HttpClient(baseClient *http.Client) *http.Client
	GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor
	GrpcServerHandler() stats.Handler
}
