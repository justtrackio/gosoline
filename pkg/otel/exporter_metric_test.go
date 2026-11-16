package otel_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/otel"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type metricReceiver struct {
	collectormetrics.UnimplementedMetricsServiceServer
	requests chan *collectormetrics.ExportMetricsServiceRequest
}

func (r *metricReceiver) Export(_ context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	r.requests <- req

	return &collectormetrics.ExportMetricsServiceResponse{}, nil
}

func TestMetricExporterMapAttributes(t *testing.T) {
	for _, protocol := range []string{otel.ProtocolGrpc, otel.ProtocolHttp} {
		t.Run(protocol, func(t *testing.T) {
			requests := make(chan *collectormetrics.ExportMetricsServiceRequest, 1)
			var endpoint string

			if protocol == otel.ProtocolGrpc {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				server := grpc.NewServer()
				collectormetrics.RegisterMetricsServiceServer(server, &metricReceiver{requests: requests})
				served := make(chan error, 1)
				go func() { served <- server.Serve(listener) }()
				t.Cleanup(func() {
					server.Stop()
					require.NoError(t, <-served)
				})
				endpoint = listener.Addr().String()
			} else {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)

						return
					}
					request := new(collectormetrics.ExportMetricsServiceRequest)
					if err := proto.Unmarshal(body, request); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)

						return
					}
					requests <- request
					w.Header().Set("Content-Type", "application/x-protobuf")
				}))
				t.Cleanup(server.Close)
				endpoint = strings.TrimPrefix(server.URL, "http://")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			exporter, err := otel.BuildMetricExporter(ctx, otel.ExporterSettings{Protocol: protocol, Endpoint: endpoint, Insecure: true, Timeout: 5 * time.Second})
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, exporter.Shutdown(context.Background())) })

			data := metricdata.ResourceMetrics{ScopeMetrics: []metricdata.ScopeMetrics{{Metrics: []metricdata.Metrics{{
				Name: "map_attribute",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.DataPoint[int64]{{
						Attributes: attribute.NewSet(attribute.Map("details", attribute.String("key", "value")), attribute.String("plain", "ordinary")),
						Value:      1,
					}},
				},
			}}}}}
			require.NoError(t, exporter.Export(ctx, &data))
			select {
			case request := <-requests:
				points := request.GetResourceMetrics()[0].GetScopeMetrics()[0].GetMetrics()[0].GetSum().GetDataPoints()
				require.Len(t, points, 1)
				attrs := points[0].GetAttributes()
				require.Len(t, attrs, 2)
				require.Equal(t, "details", attrs[0].GetKey())
				inner := attrs[0].GetValue().GetKvlistValue().GetValues()
				require.Len(t, inner, 1)
				require.Equal(t, "key", inner[0].GetKey())
				require.Equal(t, "value", inner[0].GetValue().GetStringValue())
				require.Equal(t, "plain", attrs[1].GetKey())
				require.Equal(t, "ordinary", attrs[1].GetValue().GetStringValue())
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}
