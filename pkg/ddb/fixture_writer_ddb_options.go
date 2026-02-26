package ddb

type DdbWriterOption func(settings *Settings)

func WithApplication(application string) DdbWriterOption {
	return func(settings *Settings) {
		settings.ModelId.Application = application
	}
}
