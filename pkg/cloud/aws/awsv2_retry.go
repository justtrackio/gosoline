package aws

import "github.com/aws/aws-sdk-go-v2/aws/retry"

func DefaultClientRetryOptions(settings ClientSettings) []func(*retry.StandardOptions) {
	return []func(*retry.StandardOptions){
		RetryWithMaxAttempts(settings.Backoff.MaxAttempts),
		RetryWithBackoff(NewBackoffDelayer(settings.Backoff.InitialInterval, settings.Backoff.MaxInterval)),
		RetryWithRateLimiter(NewNopRateLimiter()),
	}
}

func RetryWithMaxAttempts(maxAttempts int) func(*retry.StandardOptions) {
	return func(options *retry.StandardOptions) {
		options.MaxAttempts = maxAttempts
	}
}

func RetryWithBackoff(backoff *BackoffDelayer) func(*retry.StandardOptions) {
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
