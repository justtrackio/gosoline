package http

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

type restyClient interface {
	AddRetryCondition(conditionFunc resty.RetryConditionFunc) *resty.Client
	NewRequest() *resty.Request
	SetCookie(cookie *http.Cookie) *resty.Client
	SetCookies(cookies []*http.Cookie) *resty.Client
	SetProxy(proxy string) *resty.Client
	SetRedirectPolicy(policies ...interface{}) *resty.Client
	SetTransport(r http.RoundTripper) *resty.Client
	SetTimeout(timeout time.Duration) *resty.Client
}
