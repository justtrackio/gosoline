package grpcserver_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/grpcserver"
	protobuf "github.com/justtrackio/gosoline/pkg/grpcserver/proto/helloworld/v1"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpcServerProto "google.golang.org/grpc/health/grpc_health_v1"
)

type greeter struct {
	protobuf.UnimplementedGreeterServiceServer
}

func (g *greeter) SayHello(_ context.Context, req *protobuf.HelloRequest) (*protobuf.HelloReply, error) {
	if req.GetName() == "" {
		return nil, errors.New("empty name is not allowed")
	}

	return &protobuf.HelloReply{
		Message: fmt.Sprintf("Hello %s", req.GetName()),
	}, nil
}

func TestGRPCServer_Run_Handler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	tests := []struct {
		name    string
		defs    *grpcserver.Definitions
		reqMsg  string
		expMsg  string
		wantErr bool
	}{
		{
			name: "test handler",
			defs: &grpcserver.Definitions{
				{
					ServiceName: "greeter",
					Registrant: func(server *grpc.Server) error {
						protobuf.RegisterGreeterServiceServer(server, &greeter{})
						return nil
					},
				},
			},
			expMsg: "Hello world",
			reqMsg: "world",
		},
		{
			name: "test handler, error",
			defs: &grpcserver.Definitions{
				{
					ServiceName: "greeter",
					Registrant: func(server *grpc.Server) error {
						protobuf.RegisterGreeterServiceServer(server, &greeter{})
						return nil
					},
				},
			},
			reqMsg:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx, cancelFunc := context.WithCancel(ctx)
			defer cancelFunc()
			g, err := grpcserver.NewWithInterfaces(testCtx, logger, tt.defs, &grpcserver.Settings{Stats: grpcserver.Stats{
				Enabled:    true,
				LogPayload: false,
				LogData:    false,
				Channel:    "grpc_stats",
			}})
			assert.NoError(t, err)

			go func() {
				_ = g.Run(ctx)
			}()

			conn, err := grpc.DialContext(ctx, g.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			assert.NoError(t, err)
			defer func() {
				_ = conn.Close()
			}()

			assert.NoError(t, err)
			client := protobuf.NewGreeterServiceClient(conn)

			resp, err := client.SayHello(ctx, &protobuf.HelloRequest{Name: tt.reqMsg})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expMsg, resp.GetMessage())
		})
	}
}

func TestGRPCServer_Run_Handler_WithHealth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	tests := []struct {
		name    string
		defs    *grpcserver.Definitions
		reqMsg  string
		expMsg  string
		wantErr bool
	}{
		{
			name: "test handler with health server",
			defs: &grpcserver.Definitions{
				{
					ServiceName: "greeter",
					Registrant: func(server *grpc.Server) error {
						protobuf.RegisterGreeterServiceServer(server, &greeter{})
						return nil
					},
					HealthCheckCallback: func(ctx context.Context) grpcServerProto.HealthCheckResponse_ServingStatus {
						return grpcServerProto.HealthCheckResponse_NOT_SERVING
					},
				},
			},
			expMsg: "Hello world",
			reqMsg: "world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx, cancelFunc := context.WithCancel(ctx)
			defer cancelFunc()
			g, err := grpcserver.NewWithInterfaces(testCtx, logger, tt.defs, &grpcserver.Settings{
				Health: grpcserver.Health{
					Enabled: true,
				},
			})
			assert.NoError(t, err)

			go func() {
				_ = g.Run(ctx)
			}()

			conn, err := grpc.DialContext(ctx, g.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			assert.NoError(t, err)
			defer func() {
				_ = conn.Close()
			}()

			assert.NoError(t, err)
			client := protobuf.NewGreeterServiceClient(conn)

			resp, err := client.SayHello(ctx, &protobuf.HelloRequest{Name: tt.reqMsg})
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expMsg, resp.GetMessage())
		})
	}
}
