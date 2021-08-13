package kernel

import "time"

func KillTimeout(killTimeout time.Duration) Option {
	return func(k *kernel) error {
		k.killTimeout = killTimeout

		return nil
	}
}

func ForceExit(forceExit func(code int)) Option {
	return func(k *kernel) error {
		k.forceExit = forceExit

		return nil
	}
}
