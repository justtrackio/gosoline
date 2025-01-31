package fixtures

type FixtureSetSettings struct {
	Enabled bool
}

type FixtureSetOption func(settings *FixtureSetSettings)

func WithEnabled(enabled bool) FixtureSetOption {
	return func(settings *FixtureSetSettings) {
		settings.Enabled = enabled
	}
}

func NewFixtureSetSettings(options ...FixtureSetOption) *FixtureSetSettings {
	fso := &FixtureSetSettings{
		Enabled: true,
	}

	for _, option := range options {
		option(fso)
	}

	return fso
}
