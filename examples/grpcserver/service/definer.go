package service

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/grpcserver"
	protobuf "github.com/justtrackio/gosoline/pkg/grpcserver/proto/helloworld/v1"
	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc"
)

// Definer initializes the service dependencies, create and returns the grpcserver definitions.
func Definer(_ context.Context, config cfg.Config, logger log.Logger) (*grpcserver.Definitions, error) {
	defs := &grpcserver.Definitions{}

	// Initialize the greeter service.
	// Tip: by adding the service name here we make sure that all the logs produced by this service will contain the service name.
	greeter := NewGreeterService(config, logger.WithFields(log.Fields{
		"grpc_server": GreeterServiceName,
	}))

	// Add the greeter service definition.
	defs.AddWithHealthCheckCallback(GreeterServiceName, greeterRegistrant(greeter), greeter.CustomHealthCheck)

	return defs, nil
}

// greeterRegistrant returns a grpcserver.Registrant for the greeter service (of type protobuf.GreeterServer).
// The grpcserver.Registrant will be called by the grpcserver.Server during booting.
func greeterRegistrant(greeterServer protobuf.GreeterServiceServer) grpcserver.Registrant {
	return func(server *grpc.Server) error {
		protobuf.RegisterGreeterServiceServer(server, greeterServer)

		return nil
	}
}
