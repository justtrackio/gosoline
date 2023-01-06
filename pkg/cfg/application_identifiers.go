package cfg

import (
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

func GetAppIdFromConfig(config Config) AppId {
	return AppId{
		Project:     config.GetString("app_project"),
		Environment: config.GetString("env"),
		Family:      config.GetString("app_family"),
		Group:       config.GetString("app_group"),
		Application: config.GetString("app_name"),
	}
}

func (i *AppId) PadFromConfig(config Config) {
	if len(i.Project) == 0 {
		i.Project = config.GetString("app_project")
	}

	if len(i.Environment) == 0 {
		i.Environment = config.GetString("env")
	}

	if len(i.Family) == 0 {
		i.Family = config.GetString("app_family")
	}

	if len(i.Group) == 0 {
		i.Group = config.GetString("app_group")
	}

	if len(i.Application) == 0 {
		i.Application = config.GetString("app_name")
	}
}

func (i *AppId) String() string {
	elements := []string{i.Project, i.Environment, i.Family, i.Group, i.Application}
	elements = funk.Filter(elements, func(element string) bool {
		return len(element) > 0
	})

	return strings.Join(elements, "-")
}
