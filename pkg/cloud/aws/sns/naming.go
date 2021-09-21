package sns

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

var GetTopicName = func(appId cfg.AppId, topicId string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", appId.Project, appId.Environment, appId.Family, appId.Application, topicId)
}

func WithTopicNamingStrategy(strategy func(appId cfg.AppId, topicId string) string) {
	GetTopicName = strategy
}
