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
