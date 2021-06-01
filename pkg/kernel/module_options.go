package kernel

type ModuleOption func(ms *ModuleConfig)

// Overwrite the type a module specifies by something else.
// E.g., if you have a background module you completely depend
// on, you can do
//
// k.Add("your module", NewYourModule(), kernel.ModuleType(kernel.TypeEssential))
//
// to declare the module as essential. Now if the module quits the
// kernel will shut down instead of continuing to run.
func ModuleType(moduleTypeProvider func() TypedModule) ModuleOption {
	return func(ms *ModuleConfig) {
		moduleType := moduleTypeProvider()
		ms.Essential = moduleType.IsEssential()
		ms.Background = moduleType.IsBackground()
	}
}

// Overwrite the stage of a module. Using this, you can move a module
// of yours (or someone else) to a different stage, e.g. to make sure it
// shuts down after another module (because it is the consumer of another
// module and you need the other module to stop producing before you can
// stop consuming).
func ModuleStage(moduleStage int) ModuleOption {
	return func(ms *ModuleConfig) {
		ms.Stage = moduleStage
	}
}

// Combine a list of options by applying them in order.
func MergeOptions(options []ModuleOption) ModuleOption {
	return func(ms *ModuleConfig) {
		for _, opt := range options {
			opt(ms)
		}
	}
}
