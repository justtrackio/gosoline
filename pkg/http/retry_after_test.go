package http

import (
	"context"
	stdHttp "net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func TestRetryAfterRejectsOverBudgetBackoffWhenConfiguredMaximumIsZero(t *testing.T) {
	settings := Settings{
		RequestTimeout:     time.Minute,
		RetryWaitTime:      10 * time.Second,
		RetryMaxWaitTime:   0,
		RetryAfterSettings: RetryAfterSettings{JitterMax: 1},
	}
	// On the third attempt, Resty's fallback is at least 20s, even though
	// the configured minimum wait of 10s fits within the caller's budget.
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	client := resty.New()
	request := client.R().SetContext(ctx)
	request.Attempt = 3
	response := &resty.Response{
		Request:     request,
		RawResponse: &stdHttp.Response{StatusCode: stdHttp.StatusTooManyRequests},
	}

	_, err := newRetryAfterFunc(settings)(client, response)
	require.ErrorIs(t, err, errRetryWaitExceedsContext)
}

func TestRetryAfterAllowsRetryWithZeroEffectiveMaximum(t *testing.T) {
	for _, minWaitTime := range []time.Duration{0, time.Millisecond} {
		t.Run(minWaitTime.String(), func(t *testing.T) {
			settings := Settings{
				RetryWaitTime:      minWaitTime,
				RetryAfterSettings: RetryAfterSettings{JitterMax: 1},
			}
			client := resty.New().SetRetryAfter(newRetryAfterFunc(settings))
			request := client.R().SetContext(t.Context())
			attempts := 0
			err := resty.Backoff(func() (*resty.Response, error) {
				attempts++
				request.Attempt = attempts
				status := stdHttp.StatusOK
				if attempts == 1 {
					status = stdHttp.StatusTooManyRequests
				}

				return &resty.Response{
					Request:     request,
					RawResponse: &stdHttp.Response{StatusCode: status},
				}, nil
			}, resty.Retries(1), resty.WaitTime(settings.RetryWaitTime), resty.MaxWaitTime(getRetryMaxWaitTime(settings)),
				resty.RetryConditions([]resty.RetryConditionFunc{func(resp *resty.Response, _ error) bool {
					return resp.StatusCode() == stdHttp.StatusTooManyRequests
				}}))

			require.NoError(t, err)
			require.Equal(t, 2, attempts)
		})
	}
}
