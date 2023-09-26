package grpcserver_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/grpcserver"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	protobuf "google.golang.org/grpc/health/grpc_health_v1"
)

func Test_healthServer_Check(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	tests := []struct {
		name     string
		in       *protobuf.HealthCheckRequest
		callback *grpcserver.ServiceHealthCallback
		want     *protobuf.HealthCheckResponse
		wantErr  bool
	}{
		{
			name: "simple check",
			in:   &protobuf.HealthCheckRequest{},
			want: &protobuf.HealthCheckResponse{
				Status: protobuf.HealthCheckResponse_SERVING,
			},
		},
		{
			name: "with callback",
			in:   &protobuf.HealthCheckRequest{},
			callback: &grpcserver.ServiceHealthCallback{
				ServiceName: "test",
				HealthCheckCallback: func(ctx context.Context) protobuf.HealthCheckResponse_ServingStatus {
					return protobuf.HealthCheckResponse_SERVING
				},
			},
			want: &protobuf.HealthCheckResponse{
				Status: protobuf.HealthCheckResponse_SERVING,
			},
		},
		{
			name: "with callback, failing",
			in:   &protobuf.HealthCheckRequest{},
			callback: &grpcserver.ServiceHealthCallback{
				ServiceName: "test",
				HealthCheckCallback: func(ctx context.Context) protobuf.HealthCheckResponse_ServingStatus {
					return protobuf.HealthCheckResponse_NOT_SERVING
				},
			},
			want: &protobuf.HealthCheckResponse{
				Status: protobuf.HealthCheckResponse_NOT_SERVING,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := grpcserver.NewHealthServer(logger, cancelFunc)

			if tt.callback != nil {
				s.AddCallback(tt.callback.ServiceName, tt.callback.HealthCheckCallback)
			}

			got, err := s.Check(ctx, tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Check() got = %v, want %v", got, tt.want)
			}
		})
	}
}
