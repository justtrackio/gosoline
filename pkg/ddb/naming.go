package ddb

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
)

type NamingFactory func(modelId mdl.ModelId) string

var namingStrategy = func(modelId mdl.ModelId) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", modelId.Project, modelId.Environment, modelId.Family, modelId.Application, modelId.Name)
}

func WithNamingStrategy(strategy NamingFactory) {
	namingStrategy = strategy
}

func TableName(settings *Settings) string {
	tableName := namingStrategy(settings.ModelId)

	if settings.NamingStrategy != nil {
		tableName = settings.NamingStrategy(settings.ModelId)
	}

	return tableName
}
