# Gosoline Application Framework
![Gosoline](https://github.com/justtrackio/gosoline/workflows/Gosoline/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/justtrackio/gosoline)](https://goreportcard.com/report/github.com/justtrackio/gosoline)

Gosoline is a Golang based application framework specialized for building 
microservices in the cloud. It provides tools for handling most of the common
challenges like configuration, logging, structured code execution, handling
http requests, asynchronous message processing, writing integration tests and 
much more.

## Requirements and upgrade compatibility

Gosoline requires **Go 1.26.10 or newer**. Upgrade application builders, CI and
container toolchains together; Go 1.25 is no longer supported. A newer release
line must also include its security fixes: Go 1.27 users need at least 1.27.3.

The dependency-security update includes intentional compatibility changes:

- **CrateDB support is removed.** `db.DriverNameCrateDb` and
  `db.NewCrateDbDriver`, built-in driver registration, DSN generation and the
  CrateDB migration adapter no longer exist. Configuring `driver: cratedb`
  fails with `no driver factory defined for cratedb`. Migrate those applications
  before upgrading; MySQL and Redshift remain supported.
- **Avro uses `github.com/iskorotkov/avro/v2` instead of Hamba.** The existing
  constructor uses the fork's default codec, without an added map-allocation
  limit. Applications can opt into `stream.NewAvroEncoderWithCodec(schema, codec)`
  to supply their own limits, type registrations and converters. No Gosoline
  defaults or configuration settings are added. Update application imports and
  global registrations to the fork; reused nested-map values now merge rather
  than replace. **GO-2026-5048 remains unmitigated with the default codec, even
  though the fork version clears the scanner advisory.** See the
  [caller-supplied codec example and compatibility caveats](pkg/stream/AGENTS.md#avro-encoding).
- **S3 adopts the AWS SDK checksum defaults.** Supported uploads calculate
  CRC32 and downloads request checksum validation. S3-compatible services and
  proxies must support the resulting headers and HTTPS trailing-checksum
  encoding. Fully consume and check response-body read errors: validation can
  fail after `GetObject` succeeds. Plaintext non-seekable uploads remain
  unsupported; their error stage/message can change. Newly recognized invalid
  `AWS_REQUEST_CHECKSUM_CALCULATION` or `AWS_RESPONSE_CHECKSUM_VALIDATION`
  values can fail shared AWS configuration loading.
- **OpenTelemetry uses core 1.45 and log 0.21.** Resource schema URLs advance
  from 1.41.0 to 1.43.0. Direct consumers of the log value/attribute API must
  migrate to `otel/attribute`. Review zero log-attribute limits, environment TLS
  certificates, HTTP `Retry-After` handling and shutdown/flush errors before
  upgrading; these now follow the corrected upstream behavior.

Security fixes can also reject previously accepted malformed protocol input and
change compressed bytes, parsing, logging and authentication edge cases. The
security-update pull request records the complete dependency/version inventory,
upstream compatibility references, exercised checks and remaining limitations.

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

Read our [documentation](https://gosoline-project.github.io/docs/) to learn how to use gosoline.

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
	h.logger.Info(ctx, "Hello World")
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
go test -tags integration,fixtures ./test/...
```

### macOS

To run them on macOS you will (once initially and after each system update) have to execute this:

```shell
sudo ifconfig lo0 alias 172.17.0.1
```

Afterwards simply execute:

```shell
go test -tags integration,fixtures ./test/...
```
