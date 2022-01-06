package kernel

import "time"

func WithKillTimeout(killTimeout time.Duration) Option {
	return func(k *kernel) error {
		k.killTimeout = killTimeout

		return nil
	}
}

func WithExitHandler(handler func(code int)) Option {
	return func(k *kernel) error {
		k.exitHandler = handler

		return nil
	}
}
