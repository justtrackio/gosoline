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
	project, err := config.GetString("app_project")
	if err != nil {
		return AppId{}, fmt.Errorf("failed to get app_project: %w", err)
	}
	
	environment, err := config.GetString("env")
	if err != nil {
		return AppId{}, fmt.Errorf("failed to get env: %w", err)
	}
	
	family, err := config.GetString("app_family")
	if err != nil {
		return AppId{}, fmt.Errorf("failed to get app_family: %w", err)
	}
	
	group, err := config.GetString("app_group")
	if err != nil {
		return AppId{}, fmt.Errorf("failed to get app_group: %w", err)
	}
	
	application, err := config.GetString("app_name")
	if err != nil {
		return AppId{}, fmt.Errorf("failed to get app_name: %w", err)
	}
	
	return AppId{
		Project:     project,
		Environment: environment,
		Family:      family,
		Group:       group,
		Application: application,
	}, nil
}

func (i *AppId) PadFromConfig(config Config) error {
	if len(i.Project) == 0 {
		project, err := config.GetString("app_project")
		if err != nil {
			return fmt.Errorf("failed to get app_project: %w", err)
		}
		i.Project = project
	}

	if len(i.Environment) == 0 {
		environment, err := config.GetString("env")
		if err != nil {
			return fmt.Errorf("failed to get env: %w", err)
		}
		i.Environment = environment
	}

	if len(i.Family) == 0 {
		family, err := config.GetString("app_family")
		if err != nil {
			return fmt.Errorf("failed to get app_family: %w", err)
		}
		i.Family = family
	}

	if len(i.Group) == 0 {
		group, err := config.GetString("app_group")
		if err != nil {
			return fmt.Errorf("failed to get app_group: %w", err)
		}
		i.Group = group
	}

	if len(i.Application) == 0 {
		application, err := config.GetString("app_name")
		if err != nil {
			return fmt.Errorf("failed to get app_name: %w", err)
		}
		i.Application = application
	}
	
	return nil
}

func (i *AppId) String() string {
	elements := []string{i.Project, i.Environment, i.Family, i.Group, i.Application}
	elements = funk.Filter(elements, func(element string) bool {
		return len(element) > 0
	})

	return strings.Join(elements, "-")
}
