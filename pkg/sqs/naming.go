package sqs

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
)

type NamingFactory func(appId cfg.AppId, queueId string) string

var namingStrategy = func(appId cfg.AppId, queueId string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application, queueId)
}

func WithNamingStrategy(strategy NamingFactory) {
	namingStrategy = strategy
}
