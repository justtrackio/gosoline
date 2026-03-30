package fixtures

type FixtureSetSettings struct {
	Enabled   bool
	Shared    bool
	SharedKey string
}

type FixtureSetOption func(settings *FixtureSetSettings)

// WithEnabled toggles fixture loading for a single set.
func WithEnabled(enabled bool) FixtureSetOption {
	return func(settings *FixtureSetSettings) {
		settings.Enabled = enabled
	}
}

// WithShared marks a fixture set as shared so it is loaded once per shared
// fixture environment and skipped on subsequent loads.
func WithShared(shared bool) FixtureSetOption {
	return func(settings *FixtureSetSettings) {
		settings.Shared = shared
	}
}

// WithSharedKey overrides the default shared-fixture identity used to track
// whether a shared fixture set has already been loaded.
func WithSharedKey(sharedKey string) FixtureSetOption {
	return func(settings *FixtureSetSettings) {
		settings.SharedKey = sharedKey
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
