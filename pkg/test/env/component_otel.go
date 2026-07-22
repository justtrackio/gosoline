package env

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/test/env/otelcol"
)

type OtelComponent struct {
	baseComponent
	grpcAddress string
	httpAddress string
	client      *otelcol.Client
}

func (c *OtelComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting("otel", map[string]any{
			"exporter": map[string]any{
				"protocol": "grpc",
				"host":     c.grpcHost(),
				"port":     c.grpcPort(),
				"insecure": true,
			},
		}),
	}
}

func (c *OtelComponent) GrpcAddress() string {
	return c.grpcAddress
}

func (c *OtelComponent) HttpAddress() string {
	return c.httpAddress
}

// Client returns the OTel collector client for querying received telemetry.
func (c *OtelComponent) Client() *otelcol.Client {
	return c.client
}

func (c *OtelComponent) grpcHost() string {
	for i, ch := range c.grpcAddress {
		if ch == ':' {
			return c.grpcAddress[:i]
		}
	}

	return c.grpcAddress
}

func (c *OtelComponent) grpcPort() string {
	for i, ch := range c.grpcAddress {
		if ch == ':' {
			return c.grpcAddress[i+1:]
		}
	}

	return ""
}
