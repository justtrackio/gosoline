package cfg

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

type AppId struct {
	Project     string `cfg:"project"     default:"{app_project}"                    json:"project"`
	Environment string `cfg:"environment" default:"{env}"                            json:"environment"`
	Family      string `cfg:"family"      default:"{app_family}"                     json:"family"`
	Group       string `cfg:"group"       default:"{app_group}"                      json:"group"`
	Application string `cfg:"application" default:"{app_name}"                       json:"application"`
	Realm       string `cfg:"realm"       default:"{app_project}-{env}-{app_family}-{app_group}" json:"realm"`
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

	if appId.Realm, err = config.GetString("realm"); err != nil {
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

	if i.Realm == "" {
		if i.Realm, err = config.GetString("realm"); err != nil {
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
