package env

import (
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env/otelcol"
)

func init() {
	componentFactories[componentOtel] = &otelFactory{}
}

const componentOtel = "otel"

type otelSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	GrpcPort int `cfg:"grpc_port" default:"0"`
	HttpPort int `cfg:"http_port" default:"0"`
}

type otelFactory struct{}

func (f *otelFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("otel") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentOtel) {
		return nil
	}

	if has, err := manager.HasType(componentOtel); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	settings := &otelSettings{}
	if err := UnmarshalSettings(config, settings, componentOtel, "default"); err != nil {
		return fmt.Errorf("can not unmarshal otel settings: %w", err)
	}
	settings.Type = componentOtel

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default otel component: %w", err)
	}

	return nil
}

func (f *otelFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &otelSettings{}
}

func (f *otelFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings.(*otelSettings)),
			HealthCheck:     f.healthCheck(),
		},
	}
}

func (f *otelFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, _ any) (Component, error) {
	main := containers["main"]
	grpcBinding := main.bindings["grpc"]
	httpBinding := main.bindings["http"]

	return &OtelComponent{
		baseComponent: baseComponent{},
		grpcAddress:   fmt.Sprintf("%s:%s", grpcBinding.host, grpcBinding.port),
		httpAddress:   fmt.Sprintf("%s:%s", httpBinding.host, httpBinding.port),
		client:        otelcol.NewClient(main.name),
	}, nil
}

func (f *otelFactory) configureContainer(settings *otelSettings) *ContainerConfig {
	return &ContainerConfig{
		Auth:       settings.Image.Auth,
		Repository: settings.Image.Repository,
		Tag:        settings.Image.Tag,
		PortBindings: PortBindings{
			"grpc": {
				ContainerPort: 4317,
				HostPort:      settings.GrpcPort,
				Protocol:      "tcp",
			},
			"http": {
				ContainerPort: 4318,
				HostPort:      settings.HttpPort,
				Protocol:      "tcp",
			},
		},
	}
}

func (f *otelFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		binding := container.bindings["http"]
		url := fmt.Sprintf("http://%s:%s/", binding.host, binding.port)

		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck // health check only

		// The OTLP HTTP receiver returns 404 on / but any HTTP response means the collector is alive.
		return nil
	}
}
