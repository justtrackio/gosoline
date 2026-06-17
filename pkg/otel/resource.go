package otel

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// BuildResource constructs an OTEL resource from the application identity and the configured
// resource settings. Identity (service name/namespace, environment, extra attributes) lives in
// resource attributes — never in metric names or span names. All three signals build the resource
// from the same settings so traces, metrics, and logs share identical resource attributes
// (required for correlation).
func BuildResource(config cfg.Config, settings ResourceSettings) (*resource.Resource, error) {
	identity, err := cfg.GetAppIdentity(config)
	if err != nil {
		return nil, fmt.Errorf("could not get app identity from config: %w", err)
	}

	serviceName, err := identity.Format(settings.ServiceNamePattern, settings.Delimiter)
	if err != nil {
		return nil, fmt.Errorf("could not format service name from pattern %q: %w", settings.ServiceNamePattern, err)
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.DeploymentEnvironment(identity.Env),
	}

	if serviceNamespace, err := identity.Format(settings.ServiceNamespacePattern, settings.Delimiter); err == nil {
		attributes = append(attributes, semconv.ServiceNamespace(serviceNamespace))
	}

	for key, value := range settings.Attributes {
		formatted, err := identity.Format(value, settings.Delimiter)
		if err != nil {
			return nil, fmt.Errorf("could not format resource attribute %q value %q: %w", key, value, err)
		}

		attributes = append(attributes, attribute.String(key, formatted))
	}

	return resource.NewWithAttributes(semconv.SchemaURL, attributes...), nil
}

// ProvideResource builds the OTEL resource for the given config. It is a thin wrapper over
// BuildResource that first reads the shared settings.
func ProvideResource(config cfg.Config) (*resource.Resource, error) {
	settings, err := ReadSettings(config)
	if err != nil {
		return nil, err
	}

	return BuildResource(config, settings.Resource)
}
