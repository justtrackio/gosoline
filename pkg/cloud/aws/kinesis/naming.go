package kinesis

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type StreamNameSettingsAware interface {
	GetAppId() cfg.AppId
	GetClientName() string
	GetStreamName() string
}

type StreamNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}-{streamName}"`
}

func GetStreamName(config cfg.Config, settings StreamNameSettingsAware) (Stream, error) {
	if len(settings.GetClientName()) == 0 {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("kinesis", settings.GetClientName()))
	namingSettings := &StreamNamingSettings{}
	config.UnmarshalKey(namingKey, namingSettings)

	appId := settings.GetAppId()
	name := namingSettings.Pattern

	values := map[string]string{
		"project":    appId.Project,
		"env":        appId.Environment,
		"family":     appId.Family,
		"group":      appId.Group,
		"app":        appId.Application,
		"streamName": settings.GetStreamName(),
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		name = strings.ReplaceAll(name, templ, val)
	}

	return Stream(name), nil
}
