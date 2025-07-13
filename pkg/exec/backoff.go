package exec

import backoff "github.com/cenkalti/backoff/v4"

func NewExponentialBackOff(settings *BackoffSettings) *backoff.ExponentialBackOff {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = settings.InitialInterval
	backoffConfig.MaxInterval = settings.MaxInterval
	backoffConfig.MaxElapsedTime = settings.MaxElapsedTime

	return backoffConfig
}
