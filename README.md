# Gosoline Application Framework
![Gosoline](https://github.com/justtrackio/gosoline/workflows/Gosoline/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/justtrackio/gosoline)](https://goreportcard.com/report/github.com/justtrackio/gosoline)

Gosoline is a Golang based application framework specialized for building 
microservices in the cloud. It provides tools for handling most of the common
challenges like configuration, logging, structured code execution, handling
http requests, asynchronous message processing, writing integration tests and 
much more.

## Why use it?

- Provides an all-in-one solution for the primary use cases
- Out-of-the-box solutions for many AWS services
- Reduces a lot of needed glue and boilerplate code
- Uses a lot of sane default values to to just run code from the IDE (e.g. uses localstack instead of AWS)
- Makes it easy to build a microservice-based architecture which uses a lot of asynchronous event processing

## Key Features

- Configuration management
- Logging
- Metrics
- Tracing
- Fixture loading
- Integration tests
- Application lifecycle management
- Api server
- Consumer and producer applications
- Integrates well with AWS

## Documentation

Read our [documentation](https://justtrackio.github.io/gosoline/) to learn how to use gosoline.

## Quickstart

Every application consists of at least a main.go and config.dist.yml file.

### main.go

[embedmd]:# (examples/application/main.go)
```go
package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	application.Run(
		application.WithModuleFactory("hello-world", NewHelloWorldModule),
	)
}

func NewHelloWorldModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &HelloWorldModule{
		logger: logger.WithChannel("hello-world"),
	}, nil
}

type HelloWorldModule struct {
	logger log.Logger
}

func (h HelloWorldModule) Run(ctx context.Context) error {
	h.logger.Info("Hello World")
	return nil
}
```

### config.dist.yml

```yaml
env: dev
app_project: gosoline
app_family: example
app_name: application
```

### Output

```
14:10:10.242 main    info    applied priority 8 config post processor 'gosoline.log.handler_main'  application: application
14:10:10.242 kernel  info    starting kernel                                     application: application
14:10:10.243 kernel  info    all modules created                                 application: application
14:10:10.243 kernel  info    cfg api.health.path=/health                         application: application
14:10:10.243 kernel  info    cfg api.health.port=8090                            application: application
14:10:10.243 kernel  info    cfg app_family=example                              application: application
14:10:10.243 kernel  info    cfg app_name=application                            application: application
14:10:10.243 kernel  info    cfg app_project=gosoline                            application: application
14:10:10.243 kernel  info    cfg cfg.server.port=8070                            application: application
14:10:10.243 kernel  info    cfg env=dev                                         application: application
14:10:10.243 kernel  info    cfg kernel.killTimeout=10s                          application: application
14:10:10.243 kernel  info    cfg log.handlers.main.formatter=console             application: application
14:10:10.243 kernel  info    cfg log.handlers.main.level=info                    application: application
14:10:10.243 kernel  info    cfg log.handlers.main.timestamp_format=15:04:05.000  application: application
14:10:10.243 kernel  info    cfg log.handlers.main.type=iowriter                 application: application
14:10:10.243 kernel  info    cfg log.handlers.main.writer=stdout                 application: application
14:10:10.243 kernel  info    cfg metric.application={app_name}                   application: application
14:10:10.243 kernel  info    cfg metric.enabled=false                            application: application
14:10:10.243 kernel  info    cfg metric.environment={env}                        application: application
14:10:10.243 kernel  info    cfg metric.family={app_family}                      application: application
14:10:10.243 kernel  info    cfg metric.interval=1m0s                            application: application
14:10:10.243 kernel  info    cfg metric.project={app_project}                    application: application
14:10:10.243 kernel  info    cfg stream.metrics.messages_per_runner.enabled=false  application: application
14:10:10.243 kernel  info    cfg stream.metrics.messages_per_runner.leader_election=streamMprMetrics  application: application
14:10:10.243 kernel  info    cfg stream.metrics.messages_per_runner.max_increase_percent=200  application: application
14:10:10.244 kernel  info    cfg stream.metrics.messages_per_runner.max_increase_period=5m0s  application: application
14:10:10.244 kernel  info    cfg stream.metrics.messages_per_runner.period=1m0s  application: application
14:10:10.244 kernel  info    cfg stream.metrics.messages_per_runner.target_value=0  application: application
14:10:10.244 kernel  info    cfg fingerprint: 8df18fc41a40039f92f1f4213aee4869   application: application
14:10:10.244 kernel  info    stage 0 up and running                              application: application
14:10:10.244 kernel  info    stage 1024 up and running                           application: application
14:10:10.244 kernel  info    stage 2048 up and running                           application: application
14:10:10.244 kernel  info    kernel up and running                               application: application
14:10:10.244 kernel  info    running background module metric in stage 0         application: application
14:10:10.244 metrics info    metrics not enabled..                               application: application
14:10:10.244 kernel  info    stopped background module metric                    application: application
14:10:10.244 kernel  info    running background module api-health-check in stage 1024  application: application
14:10:10.244 kernel  info    running foreground module hello-world in stage 2048  application: application
14:10:10.244 kernel  info    running background module config-server in stage 1024  application: application
14:10:10.244 hello-world info    Hello World                                         application: application
14:10:10.244 kernel  info    stopped foreground module hello-world               application: application
14:10:10.244 config-server info    serving config on address [::]:8070                 application: application
14:10:10.244 kernel  info    stopping kernel due to: no more foreground modules in running state  application: application
14:10:10.244 kernel  info    stopping stage 2048                                 application: application
14:10:10.244 kernel  info    stopped stage 2048                                  application: application
14:10:10.244 kernel  info    stopping stage 1024                                 application: application
14:10:10.244 kernel  info    stopped background module api-health-check          application: application
14:10:10.244 kernel  info    stopped background module config-server             application: application
14:10:10.245 kernel  info    stopped stage 1024                                  application: application
14:10:10.245 kernel  info    stopping stage 0                                    application: application
14:10:10.245 kernel  info    stopped stage 0                                     application: application
14:10:10.245 kernel  info    leaving kernel                                      application: application
```

## Integration test execution

### Linux
On linux it's very straight to execute the integration tests:
```shell
go test -tags integration,fixtures ./test
```

### macOS

To run them on macOS you will (once initially and after each system update) have to execute this:

```shell
sudo ifconfig lo0 alias 172.17.0.1
```

Afterwards simply execute:

```shell
go test -tags integration,fixtures ./test
```
