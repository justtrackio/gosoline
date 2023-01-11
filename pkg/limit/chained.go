package limit

import "context"

type chained struct {
	limiters []Limiter
}

func ChainFactories(fs ...func() (Limiter, error)) (Limiter, error) {
	var limiters []Limiter
	for _, f := range fs {
		lim, err := f()
		if err != nil {
			return nil, err
		}

		limiters = append(limiters, lim)
	}

	return Chain(limiters...)
}

func Chain(ls ...Limiter) (Limiter, error) {
	return chained{ls}, nil
}

func (c chained) Wait(ctx context.Context, prefix string) error {
	for _, l := range c.limiters {
		if err := l.Wait(ctx, prefix); err != nil {
			return err
		}
	}

	return nil
}
