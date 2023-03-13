// Package limit implements various rate limiters. In the current stage (2023-03-13) limiters can only be used when
// for waiting until request limits permit operations. In the future, we might add support for limiters that can be
// used to limit incoming traffic for services.
//
// Because of limitations in the testing process of the rate limiters be aware that the limiter package is currently
// BETA and can change anytime.
package limit

import "context"

type limitPkgCtxKey string

type Factory func() (Limiter, error)

type Limiter interface {
	Wait(ctx context.Context, prefix string) error
}

type LimiterWithMiddleware interface {
	Limiter
	WithMiddleware(...MiddlewareFactory)
}

func newMiddlewareEmbeddable() *middlewareEmbeddable {
	return &middlewareEmbeddable{middleware: ChainMiddleware()}
}

type middlewareEmbeddable struct {
	middleware Middleware
}

func (d *middlewareEmbeddable) WithMiddleware(m ...MiddlewareFactory) {
	d.middleware = ChainMiddleware(m...)
}

type unlimited struct{}

func NewUnlimited() Limiter {
	return &unlimited{}
}

func (u unlimited) Wait(context.Context, string) error {
	return nil
}
