package mapx

type OpMode struct {
	IsSet        bool
	SkipExisting bool
}

type MapOption func(mode *OpMode)

func SkipExisting(mode *OpMode) {
	mode.SkipExisting = true
}
