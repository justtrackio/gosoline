## gRPC Server example tutorial

### Run instructions

run `go run main.go` to start a grpc server on port 8080

this will start the application which contains one module `grpcserver` and based on the 
`config.dist.yml` listens to port 8080

then you can use the proto definition (pkg/grpcserver/proto/helloworld/v1/helloworld.proto) 
to make some test requests (you can use a tool like [Kreya](https://kreya.app/) for that)

The default configuration also includes the health check server,
you can change this behavior by setting the `grpc_server.health.enabled` 
flag in the configuration to false.

### Settings

You can configure the gRPC server by adding the following snippet in your config file or the Environment variables

```yml
grpc_server:
  default:
      port: 8080
      health:
        enabled: true
```

### Basic usage

You have to add the gRPC Server module to your kernel

```go
app := application.New()
app.Add("grpc_server", grpcserver.New(service.Definer))
```

You have to provide a Definer which is responsible to add the services to the gRPC Server

to generate the services from the `.proto` definition you can use

```shell
protoc --go_opt=paths=source_relative --go_out=plugins=grpc:. helloworld.proto
```

tip: you can add the command to a `gen.go` which you can use by calling `go generate gen.go` command from your CLI
whenever you want to regenerate the services and the serialization/deserialization methods

example:

```go
//go:build ignore

package protobuf

//go:generate protoc --go_opt=paths=source_relative --go_out=plugins=grpc:. helloworld.proto
```

```go
func Definer(_ context.Context, config cfg.Config, logger log.Logger) (*grpcserver.Definitions, error) {
	defs := &grpcserver.Definitions{}

	greeter := NewGreeterService(config, logger)

	defs.Add(GreeterServiceName, greeterRegistrant(greeter))

	return defs, nil
}

func greeterRegistrant(greeterServer protobuf.GreeterServer) grpcserver.Registrant {
	return func(server *grpc.Server) error {
		protobuf.RegisterGreeterServer(server, greeterServer)

		return nil
	}
}
```

if you want to have a custom health status logic you can use the 
`grpcserver.Definitions.AddWithHealthCheckCallbck()` function and pass a
`grpcserver.HealthCheckCallback` as a third argument  
```go
defs.AddWithHealthCheckCallback(GreeterServiceName, greeterRegistrant(greeter), greeter.CustomHealthCheck)
```

the example service has a count that increases every time a hello request happens. After 10 requests
the health check returns NOT_SERVING and the server gracefully shuts down.