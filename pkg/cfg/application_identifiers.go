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
	Realm       string `cfg:"realm" default:"{project}-{env}-{family}-{group}" json:"realm"`
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

	// Resolve realm from config
	if err = appId.resolveRealmFromConfig(config); err != nil {
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

	// Resolve realm from config if not already set
	if i.Realm == "" {
		if err = i.resolveRealmFromConfig(config); err != nil {
			return err
		}
	}

	return nil
}

// resolveRealmFromConfig resolves the realm from global configuration
func (i *AppId) resolveRealmFromConfig(config Config) error {
	// Try to get realm pattern from global config
	globalRealmKey := "cloud.aws.realm"
	
	realmSettings := &struct {
		Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}"`
	}{}
	
	if err := config.UnmarshalKey(globalRealmKey, realmSettings); err != nil {
		return fmt.Errorf("failed to unmarshal realm settings for %s: %w", globalRealmKey, err)
	}

	// Use the AppId's ReplaceMacros method to expand the realm pattern
	i.Realm = i.ReplaceMacros(realmSettings.Pattern)
	return nil
}

// ResolveRealmPattern resolves a realm pattern from service-specific configuration with fallback to global config
func (i *AppId) ResolveRealmPattern(config Config, service string, clientName string) (string, error) {
	// Try to get realm pattern from service-specific client config first
	namingKey := fmt.Sprintf("cloud.aws.%s.clients.%s.naming.realm", service, clientName)
	
	// Fall back to service-specific default client config
	defaultPatternKey := fmt.Sprintf("cloud.aws.%s.clients.default.naming.realm", service)
	
	// Fall back to global realm config
	globalRealmKey := "cloud.aws.realm"
	
	realmSettings := &struct {
		Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}"`
	}{}
	
	// Try to unmarshal with fallback chain
	if err := config.UnmarshalKey(namingKey, realmSettings, 
		UnmarshalWithDefaultsFromKey(globalRealmKey, "."),
		UnmarshalWithDefaultsFromKey(defaultPatternKey, ".")); err != nil {
		return "", fmt.Errorf("failed to unmarshal realm settings for %s: %w", namingKey, err)
	}

	// Use the AppId's ReplaceMacros method to expand the realm pattern
	return i.ReplaceMacros(realmSettings.Pattern), nil
}

func (i *AppId) String() string {
	elements := []string{i.Project, i.Environment, i.Family, i.Group, i.Application, i.Realm}
	elements = funk.Filter(elements, func(element string) bool {
		return element != ""
	})

	return strings.Join(elements, "-")
}

// ReplaceMacros replaces macros in a pattern with AppId values and additional macro values
// Extra macros are replaced once before and once after the AppId macros
func (i *AppId) ReplaceMacros(pattern string, extraMacros ...MacroValue) string {
	result := pattern
	
	// First pass: replace extra macros
	for _, mv := range extraMacros {
		templ := fmt.Sprintf("{%s}", mv.Macro)
		result = strings.ReplaceAll(result, templ, mv.Value)
	}
	
	// Replace AppId macros (including realm first)
	allMacros := []MacroValue{
		{"realm", i.Realm},
		{"project", i.Project},
		{"env", i.Environment},
		{"family", i.Family},
		{"group", i.Group},
		{"app", i.Application},
	}
	
	// Append extra macros to replace them after AppId macros
	allMacros = append(allMacros, extraMacros...)
	
	for _, mv := range allMacros {
		templ := fmt.Sprintf("{%s}", mv.Macro)
		result = strings.ReplaceAll(result, templ, mv.Value)
	}
	
	return result
}

// MacroValue represents a macro and its replacement value
type MacroValue struct {
	Macro string
	Value string
}






