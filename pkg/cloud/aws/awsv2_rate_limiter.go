package aws

import "context"

type NopRateLimiter struct{}

func NewNopRateLimiter() NopRateLimiter {
	return NopRateLimiter{}
}

func (n NopRateLimiter) GetToken(_ context.Context, _ uint) (releaseToken func() error, err error) {
	return alwaysSucceed, nil
}

func (n NopRateLimiter) AddTokens(_ uint) error {
	return nil
}

func alwaysSucceed() error {
	return nil
}
