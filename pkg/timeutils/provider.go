package timeutils

import "time"

//go:generate mockery -name TimeProvider
type TimeProvider interface {
	Now() time.Time
}

var globalTimeProvider TimeProvider = systemTimeProvider{}

type systemTimeProvider struct{}

func (systemTimeProvider) Now() time.Time {
	return time.Now()
}

type timeProviderFunc struct {
	now func() time.Time
}

func (t timeProviderFunc) Now() time.Time {
	return t.now()
}

func TimeProviderFunc(f func() time.Time) TimeProvider {
	return timeProviderFunc{
		now: f,
	}
}

func ConstantTimeProvider(t time.Time) TimeProvider {
	return timeProviderFunc{
		now: func() time.Time {
			return t
		},
	}
}

func SetGlobalTimeProvider(provider TimeProvider) {
	if provider == nil {
		provider = systemTimeProvider{}
	}

	globalTimeProvider = provider
}

func Now() time.Time {
	return globalTimeProvider.Now()
}
