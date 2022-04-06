package main

import (
	"github.com/justtrackio/gosoline/examples/grpcserver/service"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/grpcserver"
)

// Example usage of the grpcserver module

func main() {
	// initialize the application
	app := application.New(
		application.WithConfigFile("./config.dist.yml", "yml"), // read config form config.dist.yml file
		application.WithLoggerHandlersFromConfig,               // enable logging based on config
	)

	// initialize the grpcserver kernel.ModuleFactory by passing a grpcserver.ServiceDefiner function to it
	grpcServerModule := grpcserver.New("default", service.Definer)
	grpcServerModuleWithoutHealthChecks := grpcserver.New("no_health_checks", service.Definer)

	// add the grpcserver kernel.ModuleFactory to the kernel
	app.Add("grpc_server", grpcServerModule)
	app.Add("grpc_server_no_health_checks", grpcServerModuleWithoutHealthChecks)

	// start the app
	app.Run()
}
