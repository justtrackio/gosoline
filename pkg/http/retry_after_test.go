package http

import (
	"context"
	stdHttp "net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func TestRetryAfterBudget(t *testing.T) {
	const minWaitTime = 10 * time.Second
	const budget = 15 * time.Second

	for _, tc := range []struct {
		name            string
		status          int
		header          string
		defaultWaitTime time.Duration
		configuredMax   time.Duration
		budget          time.Duration
		wantWait        time.Duration
	}{
		{name: "zero Retry-After uses Resty fallback", status: stdHttp.StatusServiceUnavailable, header: "0", configuredMax: minWaitTime, budget: budget},
		{name: "503 default exceeds budget", status: stdHttp.StatusServiceUnavailable, defaultWaitTime: 30 * time.Second, budget: budget},
		{name: "503 minimum exceeds budget", status: stdHttp.StatusServiceUnavailable, defaultWaitTime: time.Second, budget: 5 * time.Second},
		{name: "ordinary backoff exceeds budget", status: stdHttp.StatusTooManyRequests, configuredMax: 40 * time.Second, budget: budget},
		{name: "ordinary backoff uses configured cap", status: stdHttp.StatusTooManyRequests, configuredMax: minWaitTime, budget: budget, wantWait: minWaitTime},
		{name: "zero configured cap uses Resty fallback", status: stdHttp.StatusTooManyRequests, budget: budget},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := Settings{
				RequestTimeout:   time.Minute,
				RetryWaitTime:    minWaitTime,
				RetryMaxWaitTime: tc.configuredMax,
				RetryAfterSettings: RetryAfterSettings{
					DefaultWaitTime: tc.defaultWaitTime,
					JitterMin:       1,
					JitterMax:       1,
				},
			}
			client := resty.New().SetRetryWaitTime(settings.RetryWaitTime).SetRetryMaxWaitTime(getRetryMaxWaitTime(settings))
			ctx, cancel := context.WithTimeout(t.Context(), tc.budget)
			defer cancel()

			request := client.R().SetContext(ctx)
			// The third attempt's fallback is at least 20s, even though the 10s minimum fits.
			request.Attempt = 3
			rawResponse := &stdHttp.Response{StatusCode: tc.status, Header: make(stdHttp.Header)}
			if tc.header != "" {
				rawResponse.Header.Set("Retry-After", tc.header)
			}

			wait, err := newRetryAfterFunc(settings)(client, &resty.Response{Request: request, RawResponse: rawResponse})
			if tc.wantWait == 0 {
				require.ErrorIs(t, err, errRetryWaitExceedsContext)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantWait, wait)
		})
	}
}

func TestRetryAfterAllowsRetryWithZeroEffectiveMaximum(t *testing.T) {
	for _, minWaitTime := range []time.Duration{0, time.Millisecond} {
		t.Run(minWaitTime.String(), func(t *testing.T) {
			settings := Settings{
				RetryWaitTime:      minWaitTime,
				RetryAfterSettings: RetryAfterSettings{JitterMax: 1},
			}
			client := resty.New().
				SetRetryWaitTime(settings.RetryWaitTime).
				SetRetryMaxWaitTime(getRetryMaxWaitTime(settings)).
				SetRetryAfter(newRetryAfterFunc(settings))
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
			}, resty.Retries(1), resty.WaitTime(client.RetryWaitTime), resty.MaxWaitTime(client.RetryMaxWaitTime),
				resty.RetryConditions([]resty.RetryConditionFunc{func(resp *resty.Response, _ error) bool {
					return resp.StatusCode() == stdHttp.StatusTooManyRequests
				}}))

			require.NoError(t, err)
			require.Equal(t, 2, attempts)
		})
	}
}
