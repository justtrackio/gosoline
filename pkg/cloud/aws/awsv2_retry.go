package aws

import "github.com/aws/aws-sdk-go-v2/aws/retry"

func DefaultClientRetryOptions(clientConfig ClientConfigAware) []func(*retry.StandardOptions) {
	settings := clientConfig.GetSettings()

	options := []func(*retry.StandardOptions){
		RetryWithMaxAttempts(settings.Backoff.MaxAttempts),
		RetryWithBackoff(NewBackoffDelayer(settings.Backoff.InitialInterval, settings.Backoff.MaxInterval)),
		RetryWithRateLimiter(NewNopRateLimiter()),
	}

	options = append(options, clientConfig.GetRetryOptions()...)

	return options
}

func RetryWithMaxAttempts(maxAttempts int) func(*retry.StandardOptions) {
	return func(options *retry.StandardOptions) {
		options.MaxAttempts = maxAttempts
	}
}

func RetryWithBackoff(backoff retry.BackoffDelayer) func(*retry.StandardOptions) {
	return func(options *retry.StandardOptions) {
		options.Backoff = backoff
	}
}

func RetryWithRateLimiter(rateLimiter retry.RateLimiter) func(options *retry.StandardOptions) {
	return func(options *retry.StandardOptions) {
		options.RateLimiter = rateLimiter
	}
}

func RetryWithRetryables(retryables []retry.IsErrorRetryable) func(options *retry.StandardOptions) {
	return func(options *retry.StandardOptions) {
		options.Retryables = append(options.Retryables, retryables...)
	}
}
