package cfg

import "fmt"

type AppId struct {
	Project     string `cfg:"project"`
	Environment string `cfg:"environment"`
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
}

func GetAppIdFromConfig(config Config) AppId {
	return AppId{
		Project:     config.GetString("app_project"),
		Environment: config.GetString("env"),
		Family:      config.GetString("app_family"),
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

	if len(i.Application) == 0 {
		i.Application = config.GetString("app_name")
	}
}

func (i *AppId) String() string {
	return fmt.Sprintf("%s-%s-%s-%s", i.Project, i.Environment, i.Family, i.Application)
}
