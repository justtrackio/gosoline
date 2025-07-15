package aws

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type RealmSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}"`
}

// ResolveRealm resolves the realm pattern from configuration and expands it with the provided AppId
func ResolveRealm(config cfg.Config, appId cfg.AppId, service string, clientName string) (string, error) {
	// Try to get realm pattern from service-specific client config first
	namingKey := fmt.Sprintf("%s.naming.realm", GetClientConfigKey(service, clientName))
	
	// Fall back to service-specific default client config
	defaultPatternKey := fmt.Sprintf("%s.naming.realm", GetClientConfigKey(service, "default"))
	
	// Fall back to global realm config
	globalRealmKey := "cloud.aws.realm"
	
	realmSettings := &RealmSettings{}
	
	// Try to unmarshal with fallback chain
	if err := config.UnmarshalKey(namingKey, realmSettings, 
		cfg.UnmarshalWithDefaultsFromKey(globalRealmKey, "."),
		cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, ".")); err != nil {
		return "", fmt.Errorf("failed to unmarshal realm settings for %s: %w", namingKey, err)
	}

	// Expand the realm pattern with appId values
	name := realmSettings.Pattern
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		name = strings.ReplaceAll(name, templ, val)
	}

	return name, nil
}