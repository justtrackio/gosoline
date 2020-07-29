package cfg

type mapMode struct {
	isSet        bool
	skipExisting bool
}

type MapOption func(mode *mapMode)

func SkipExisting(mode *mapMode) {
	mode.skipExisting = true
}
