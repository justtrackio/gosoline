package fixtures

type FixtureSetSettings struct {
	Enabled bool
	Purge   bool
}

type FixtureSetOption func(settings *FixtureSetSettings)

func WithEnabled(enabled bool) FixtureSetOption {
	return func(settings *FixtureSetSettings) {
		settings.Enabled = enabled
	}
}

func WithPurge(purge bool) FixtureSetOption {
	return func(settings *FixtureSetSettings) {
		settings.Purge = purge
	}
}

func NewFixtureSetSettings(options ...FixtureSetOption) *FixtureSetSettings {
	fso := &FixtureSetSettings{
		Enabled: true,
		Purge:   false,
	}

	for _, option := range options {
		option(fso)
	}

	return fso
}
