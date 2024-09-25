package grpcserver_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/grpcserver"
	"github.com/stretchr/testify/assert"
	protobuf "google.golang.org/grpc/health/grpc_health_v1"
)

func TestDefinitions_Add(t *testing.T) {
	type args struct {
		name       string
		registrant grpcserver.Registrant
	}
	tests := []struct {
		name string
		s    grpcserver.Definitions
		args args
		want *grpcserver.Definitions
	}{
		{
			name: "new",
			s:    grpcserver.Definitions{},
			args: args{
				name: "s1",
			},
			want: &grpcserver.Definitions{
				{
					ServiceName: "s1",
				},
			},
		},
		{
			name: "append",
			s: grpcserver.Definitions{
				{
					ServiceName: "s0",
				},
			},
			args: args{
				name: "s1",
			},
			want: &grpcserver.Definitions{
				{
					ServiceName: "s0",
				},
				{
					ServiceName: "s1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Add(tt.args.name, tt.args.registrant); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefinitions_AddWithHealthCheckCallback(t *testing.T) {
	type args struct {
		name                string
		registrant          grpcserver.Registrant
		healthCheckCallback grpcserver.HealthCheckCallback
	}
	tests := []struct {
		name             string
		s                grpcserver.Definitions
		args             args
		wantServiceName  string
		wantHealthStatus protobuf.HealthCheckResponse_ServingStatus
	}{
		{
			name: "new with health callback",
			s:    grpcserver.Definitions{},
			args: args{
				name:                "s1",
				healthCheckCallback: getMockedHealthCheckCallback(protobuf.HealthCheckResponse_SERVING),
			},
			wantServiceName:  "s1",
			wantHealthStatus: protobuf.HealthCheckResponse_SERVING,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defs := tt.s.AddWithHealthCheckCallback(tt.args.name, tt.args.registrant, tt.args.healthCheckCallback)

			assert.Equal(t, tt.args.name, (*defs)[0].ServiceName)
			assert.Equal(t, (*defs)[0].HealthCheckCallback(context.Background()), protobuf.HealthCheckResponse_SERVING)
		})
	}
}

func getMockedHealthCheckCallback(fixedResponse protobuf.HealthCheckResponse_ServingStatus) grpcserver.HealthCheckCallback {
	return func(ctx context.Context) protobuf.HealthCheckResponse_ServingStatus {
		return fixedResponse
	}
}
