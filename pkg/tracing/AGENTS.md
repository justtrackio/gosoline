# Tracing Package Agent Guide

## Scope
- Distributed tracing via AWS X-Ray and OpenTelemetry.
- Provides `Tracer`, `Span` interfaces, and context propagation helpers.
- Handles sampling, context-missing strategies, and instrumentation (HTTP/GRPC).

## Key files
- `tracer.go` - Main interface and factory.
- `tracer_aws.go` - AWS X-Ray implementation.
- `tracer_otel.go` - OpenTelemetry implementation.
- `span.go` / `span_otel.go` - Span implementations.
- `instrumentor*.go` - Middleware for HTTP/GRPC instrumentation.
- `naming.go` - Naming pattern expansion logic.

## Configuration
Tracing is configured via the `tracing` key.

### Common settings
```yaml
tracing:
  provider: xray # xray, otel, local, noop
  naming:
    # Pattern for service name / appId.
    # Supported placeholders: {app.env}, {app.name}, {app.tags.<key>}
    pattern: "{app.namespace}-{app.name}"
```

### X-Ray specific
```yaml
tracing.xray:
  # Address of the X-Ray daemon
  addr_type: local # local, dns_srv
  add_value: "" # if empty and dns_srv, uses srv_pattern
  srv_pattern: "xray.{app.env}.{app.tags.family}"
  
  sampling:
    default:
      fixed_target: 1
      rate: 0.1
```

### OpenTelemetry specific
```yaml
tracing.otel:
  exporter: otel_grpc # otel_http (legacy), otel_grpc, stdout
  sampling_ratio: 0.05
  propagators: [tracecontext, baggage] # W3C trace context and baggage
```

- `otel_grpc` and `stdout` exporters use the shared `otel.*` config (resource + exporter, incl.
  TLS/mTLS); see `pkg/otel`. The legacy `otel_http` exporter reads `tracing.otel.http.*`.
- The OTEL resource (service.name/namespace, deployment.environment) is built by `pkg/otel` so
  traces share identical resource attributes with metrics and logs for correlation.

## Naming Pattern
Tracing uses a naming pattern system that delegates to `cfg.Identity.Format()` for placeholder expansion.
- Placeholders: `{app.env}`, `{app.name}`, `{app.namespace}`, `{app.tags.<key>}`.
- All referenced tags **must** be present in `app.tags`.
- If a tag is missing, initialization fails with a clear error.
- Default pattern: `{app.namespace}-{app.name}`.

## Common tasks
- Add new provider: implement `Tracer` interface, register in `tracer.go`.
- Add new instrumentor: implement `Instrumentor` interface, register in `tracer.go`.
- Adjust sampling: modify `sampling.go` or provider-specific sampling logic.

## Testing
- `go test ./pkg/tracing` covers basic logic.
- Integration tests in `test/` often use `tracing.NewLocalTracer()` or `tracing.NewNoopTracer()`.
