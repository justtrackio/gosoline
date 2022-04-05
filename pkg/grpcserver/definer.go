package grpcserver

import (
	"context"

	"google.golang.org/grpc"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// ServiceDefiner is used to initialise the Server Module.
//
// This function is used to initialise your dependencies i.e. the services.
type ServiceDefiner func(ctx context.Context, config cfg.Config, logger log.Logger) (*Definitions, error)

// Registrant is a function callback that receives a cancelFunc and a grpc.Server and should be implemented
// to add the gRPC services to the grpc.Server.
type Registrant func(server *grpc.Server) error

type definition struct {
	ServiceName         string
	Registrant          Registrant
	HealthCheckCallback HealthCheckCallback
}

// Definitions is a collection that is used to initialize the Server.
//
// During the initialisation the Registrant function will be called.
type Definitions []definition

// Add a new definition by providing the service name and the registrant.
func (s *Definitions) Add(name string, registrant Registrant) *Definitions {
	*s = append(*s, definition{
		ServiceName: name,
		Registrant:  registrant,
	})

	return s
}

// AddWithHealthCheckCallback a new definition by providing the service name, the registrant and the HealthCheckCallback.
func (s *Definitions) AddWithHealthCheckCallback(name string, registrant Registrant, healthCheckCallback HealthCheckCallback) *Definitions {
	*s = append(*s, definition{
		ServiceName:         name,
		Registrant:          registrant,
		HealthCheckCallback: healthCheckCallback,
	})

	return s
}
