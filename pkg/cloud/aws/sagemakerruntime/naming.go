package sagemakerruntime

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type EndpointNameSettingsAware interface {
	GetIdentity() cfg.Identity
	GetClientName() string
}

type EndpointNameSettings struct {
	Identity   cfg.Identity
	ClientName string
}

func (s EndpointNameSettings) GetIdentity() cfg.Identity {
	return s.Identity
}

func (s EndpointNameSettings) GetClientName() string {
	return s.ClientName
}

type EndpointNamingSettings struct {
	EndpointPattern   string `cfg:"endpoint_pattern,nodecode" default:"{app.namespace}-{name}"`
	EndpointDelimiter string `cfg:"endpoint_delimiter" default:"-"`
	Name              string `cfg:"name"`
}

func GetEndpointName(config cfg.Config, endpointSettings EndpointNameSettingsAware) (string, error) {
	if endpointSettings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("sagemakerruntime", endpointSettings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.endpoint_pattern", aws.GetClientConfigKey("sagemakerruntime", "default"))
	namingSettings := &EndpointNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "endpoint_pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal sagemakerruntime naming settings for %s: %w", namingKey, err)
	}

	identity := endpointSettings.GetIdentity()
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.EndpointPattern, namingSettings.EndpointDelimiter, map[string]string{
		"name": namingSettings.Name,
	})
	if err != nil {
		return "", fmt.Errorf("sagemakerruntime endpoint naming failed: %w", err)
	}

	return name, nil
}
