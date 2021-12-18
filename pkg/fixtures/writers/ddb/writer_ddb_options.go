package ddb

import "github.com/justtrackio/gosoline/pkg/ddb"

type DdbWriterOption func(settings *ddb.Settings)

func WithDdbModelIdApplication(application string) DdbWriterOption {
	return func(settings *ddb.Settings) {
		settings.ModelId.Application = application
	}
}
