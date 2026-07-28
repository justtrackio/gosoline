package env

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/test/env/otelcol"
)

type OtelComponent struct {
	baseComponent
	protocol        string
	exporterBinding ContainerBinding
	grpcAddress     string
	httpAddress     string
	client          *otelcol.Client
}

func (c *OtelComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{
		cfg.WithConfigSetting("otel", map[string]any{
			"exporter": map[string]any{
				"protocol": c.protocol,
				"host":     c.exporterBinding.host,
				"port":     c.exporterBinding.port,
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
