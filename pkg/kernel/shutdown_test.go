package kernel

// WithShutdownHandlerForTest replaces the kernel's shutdown handlers for tests.
func WithShutdownHandlerForTest(handler ShutdownHandler) Option {
	return func(bp *blueprint) {
		bp.kernelOptions = append(bp.kernelOptions, func(k *kernel) {
			k.shutdownHandlers = []ShutdownHandler{handler}
		})
	}
}
