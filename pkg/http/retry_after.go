package http

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	stdHttp "net/http"
	"strconv"
	"strings"
	"time"

	httpHeaders "github.com/go-http-utils/headers"
	"github.com/go-resty/resty/v2"
)

// RetryAfterSettings configures how the HTTP client handles Retry-After headers on retried responses.
type RetryAfterSettings struct {
	DefaultWaitTime time.Duration `cfg:"default_wait_time" default:"1s" validate:"min=0"`
	JitterMin       float64       `cfg:"jitter_min" default:"1" validate:"min=1"`
	JitterMax       float64       `cfg:"jitter_max" default:"1.5" validate:"min=1"`
}

func newRetryAfterFunc(settings Settings) resty.RetryAfterFunc {
	return func(_ *resty.Client, response *resty.Response) (time.Duration, error) {
		if response == nil || response.Request == nil {
			return 0, nil
		}

		retryAfterHeader := strings.TrimSpace(response.Header().Get(httpHeaders.RetryAfter))
		if retryAfterHeader == "" && response.StatusCode() != stdHttp.StatusServiceUnavailable {
			retryAfter := defaultRetryWaitTime(settings, response.Request.Attempt)
			if err := ensureRetryWaitFitsContext(response.Request.Context(), retryAfter); err != nil {
				return 0, err
			}

			return retryAfter, nil
		}

		retryAfter, err := parseRetryAfter(retryAfterHeader, time.Now())
		if err != nil {
			retryAfter = settings.RetryAfterSettings.DefaultWaitTime
		}

		retryAfter = jitterRetryAfter(retryAfter, settings.RetryAfterSettings.JitterMin, settings.RetryAfterSettings.JitterMax)

		if err := ensureRetryWaitFitsContext(response.Request.Context(), retryAfter); err != nil {
			return 0, err
		}

		return retryAfter, nil
	}
}

func defaultRetryWaitTime(settings Settings, attempt int) time.Duration {
	minWaitTime := settings.RetryWaitTime
	maxWaitTime := settings.RetryMaxWaitTime

	if minWaitTime <= 0 || maxWaitTime <= 0 {
		return 0
	}

	// Resty increments Request.Attempt before executing. Its backoff receives a zero-based retry attempt.
	attempt = max(0, attempt-1)

	temp := min(float64(maxWaitTime), float64(minWaitTime)*math.Exp2(float64(attempt)))
	center := max(time.Duration(temp/2), time.Nanosecond)
	waitTime := center + time.Duration(rand.Int63n(int64(center)))

	return max(minWaitTime, waitTime)
}

func parseRetryAfter(headerValue string, now time.Time) (time.Duration, error) {
	seconds, err := strconv.ParseInt(headerValue, 10, 64)
	if err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("retry-after seconds must not be negative")
		}

		return time.Duration(seconds) * time.Second, nil
	}

	retryAt, err := stdHttp.ParseTime(headerValue)
	if err != nil {
		return 0, fmt.Errorf("invalid retry-after header %q: %w", headerValue, err)
	}

	return max(0, retryAt.Sub(now)), nil
}

func jitterRetryAfter(retryAfter time.Duration, jitterMin float64, jitterMax float64) time.Duration {
	if retryAfter <= 0 {
		return 0
	}

	jitter := jitterMin + rand.Float64()*(max(jitterMin, jitterMax)-jitterMin)

	return time.Duration(float64(retryAfter) * jitter)
}

func ensureRetryWaitFitsContext(ctx context.Context, retryAfter time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}

	remaining := time.Until(deadline)
	if retryAfter <= remaining {
		return nil
	}

	return fmt.Errorf("retry wait duration %s exceeds remaining request time %s", retryAfter, remaining)
}

func getRetryMaxWaitTime(settings Settings) time.Duration {
	maxWaitTime := max(settings.RetryMaxWaitTime, settings.RequestTimeout, settings.RetryAfterSettings.DefaultWaitTime)
	jitterMax := settings.RetryAfterSettings.JitterMax

	return time.Duration(float64(maxWaitTime) * jitterMax)
}
