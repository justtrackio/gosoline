package http

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type CircuitBreakerSettings struct {
	Enabled          bool          `cfg:"enabled" default:"false"`
	MaxFailures      int64         `cfg:"max_failures" default:"10"`
	RetryDelay       time.Duration `cfg:"retry_delay" default:"1m"`
	ExpectedStatuses []int         `cfg:"expected_statuses"`
}

type CircuitIsOpenError struct{}

func (c CircuitIsOpenError) Error() string {
	return "request rejected, circuit breaker is open"
}

type circuitBreakerClient struct {
	Client
	logger   log.Logger
	clock    clock.Clock
	name     string
	settings CircuitBreakerSettings

	// state updated with atomics
	recentFailures int64
	nextRetryAt    int64
}

func NewCircuitBreakerClientWithInterfaces(baseClient Client, logger log.Logger, clock clock.Clock, name string, settings CircuitBreakerSettings) Client {
	return &circuitBreakerClient{
		Client:         baseClient,
		logger:         logger.WithChannel("circuit-breaker-client-" + name),
		clock:          clock,
		name:           name,
		settings:       settings,
		recentFailures: 0,
		nextRetryAt:    0,
	}
}

func (c *circuitBreakerClient) Delete(ctx context.Context, request *Request) (*Response, error) {
	return c.doRequest(ctx, request, c.Client.Delete)
}

func (c *circuitBreakerClient) Get(ctx context.Context, request *Request) (*Response, error) {
	return c.doRequest(ctx, request, c.Client.Get)
}

func (c *circuitBreakerClient) Patch(ctx context.Context, request *Request) (*Response, error) {
	return c.doRequest(ctx, request, c.Client.Patch)
}

func (c *circuitBreakerClient) Post(ctx context.Context, request *Request) (*Response, error) {
	return c.doRequest(ctx, request, c.Client.Post)
}

func (c *circuitBreakerClient) Put(ctx context.Context, request *Request) (*Response, error) {
	return c.doRequest(ctx, request, c.Client.Put)
}

func (c *circuitBreakerClient) doRequest(ctx context.Context, request *Request, performRequest func(ctx context.Context, request *Request) (*Response, error)) (*Response, error) {
	if c.isCircuitOpen(ctx) {
		return nil, CircuitIsOpenError{}
	}

	// perform the request and reset the failure counter should we succeed
	response, err := performRequest(ctx, request)
	if !c.isRemoteFailure(response, err) {
		// only reset the counter if the request was successful (ignore e.g. context canceled errors, they are not successful)
		if !exec.IsRequestCanceled(err) {
			oldFailures := atomic.SwapInt64(&c.recentFailures, 0)
			if oldFailures > 0 {
				c.logger.Info(ctx, "reset failure counter of circuit breaker again")
			}
		}

		return response, err
	}

	// ensure we at most retry after the retry delay
	newNextRetryAt := c.clock.Now().Add(c.settings.RetryDelay).UnixMilli()
	atomic.StoreInt64(&c.nextRetryAt, newNextRetryAt)

	// only now count up the recent failures - so nextRetryAt is set up already when we check it
	newFailures := atomic.AddInt64(&c.recentFailures, 1)
	if newFailures == c.settings.MaxFailures {
		c.logger.Warn(ctx, "circuit breaker triggered, stopping requests for %v", c.settings.RetryDelay)
	}

	return response, err
}

func (c *circuitBreakerClient) isCircuitOpen(ctx context.Context) bool {
	recentFailures := atomic.LoadInt64(&c.recentFailures)
	if recentFailures < c.settings.MaxFailures {
		return false
	}

	// we have too many failures. check if we can retry anyway
	nextRetryAt := atomic.LoadInt64(&c.nextRetryAt)
	now := c.clock.Now().UnixMilli()
	if nextRetryAt > now {
		return true
	}

	// we can retry. swap the retry value with the next time we can retry. if we lose the race with
	// another thread, we will not swap and must not retry
	canRetry := atomic.CompareAndSwapInt64(&c.nextRetryAt, nextRetryAt, now+c.settings.RetryDelay.Milliseconds())
	if canRetry {
		c.logger.Info(ctx, "trying to close circuit breaker again by trying single request")
	}

	return !canRetry
}

func (c *circuitBreakerClient) isRemoteFailure(response *Response, err error) bool {
	if err != nil {
		return !exec.IsRequestCanceled(err)
	}

	if len(c.settings.ExpectedStatuses) == 0 {
		return false
	}

	return !funk.Contains(c.settings.ExpectedStatuses, response.StatusCode)
}
