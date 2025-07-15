package cfg

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

type AppId struct {
	Project     string `cfg:"project" default:"{app_project}" json:"project"`
	Environment string `cfg:"environment" default:"{env}" json:"environment"`
	Family      string `cfg:"family" default:"{app_family}" json:"family"`
	Group       string `cfg:"group" default:"{app_group}" json:"group"`
	Application string `cfg:"application" default:"{app_name}" json:"application"`
}

func GetAppIdFromConfig(config Config) (AppId, error) {
	var err error
	appId := AppId{}

	if appId.Project, err = config.GetString("app_project"); err != nil {
		return appId, err
	}

	if appId.Environment, err = config.GetString("env"); err != nil {
		return appId, err
	}

	if appId.Family, err = config.GetString("app_family"); err != nil {
		return appId, err
	}

	if appId.Group, err = config.GetString("app_group"); err != nil {
		return appId, err
	}

	if appId.Application, err = config.GetString("app_name"); err != nil {
		return appId, err
	}

	return appId, nil
}

func (i *AppId) PadFromConfig(config Config) error {
	var err error

	if i.Project == "" {
		if i.Project, err = config.GetString("app_project"); err != nil {
			return err
		}
	}

	if i.Environment == "" {
		if i.Environment, err = config.GetString("env"); err != nil {
			return err
		}
	}

	if i.Family == "" {
		if i.Family, err = config.GetString("app_family"); err != nil {
			return err
		}
	}

	if i.Group == "" {
		if i.Group, err = config.GetString("app_group"); err != nil {
			return err
		}
	}

	if i.Application == "" {
		if i.Application, err = config.GetString("app_name"); err != nil {
			return err
		}
	}

	return nil
}

func (i *AppId) String() string {
	elements := []string{i.Project, i.Environment, i.Family, i.Group, i.Application}
	elements = funk.Filter(elements, func(element string) bool {
		return element != ""
	})

	return strings.Join(elements, "-")
}

type RealmSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}"`
}

// ResolveRealm resolves the realm pattern from configuration and expands it with the provided AppId
func ResolveRealm(config Config, appId AppId, service string, clientName string) (string, error) {
	// Try to get realm pattern from service-specific client config first
	namingKey := fmt.Sprintf("cloud.aws.%s.clients.%s.naming.realm", service, clientName)
	
	// Fall back to service-specific default client config
	defaultPatternKey := fmt.Sprintf("cloud.aws.%s.clients.default.naming.realm", service)
	
	// Fall back to global realm config
	globalRealmKey := "cloud.aws.realm"
	
	realmSettings := &RealmSettings{}
	
	// Try to unmarshal with fallback chain
	if err := config.UnmarshalKey(namingKey, realmSettings, 
		UnmarshalWithDefaultsFromKey(globalRealmKey, "."),
		UnmarshalWithDefaultsFromKey(defaultPatternKey, ".")); err != nil {
		return "", fmt.Errorf("failed to unmarshal realm settings for %s: %w", namingKey, err)
	}

	// Expand the realm pattern with appId values
	values := []MacroValue{
		{"project", appId.Project},
		{"env", appId.Environment},
		{"family", appId.Family},
		{"group", appId.Group},
		{"app", appId.Application},
	}

	return ReplaceMacros(realmSettings.Pattern, values), nil
}

// MacroValue represents a macro and its replacement value
type MacroValue struct {
	Macro string
	Value string
}

// ReplaceMacros replaces macros in a string with their values
// The slice is processed in order, allowing realm to be resolved first
func ReplaceMacros(pattern string, values []MacroValue) string {
	result := pattern
	for _, mv := range values {
		templ := fmt.Sprintf("{%s}", mv.Macro)
		result = strings.ReplaceAll(result, templ, mv.Value)
	}
	return result
}
