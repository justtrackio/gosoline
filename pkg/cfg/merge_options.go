package cfg

import "github.com/justtrackio/gosoline/pkg/mapx"

type MergeOption func() func(mode *mapx.OpMode)

func SkipExisting() func(mode *mapx.OpMode) {
	return func(mode *mapx.OpMode) {
		mode.SkipExisting = true
	}
}

func mergeToMapOptions(mergeOptions []MergeOption) []mapx.MapOption {
	mapOptions := make([]mapx.MapOption, len(mergeOptions))

	for i := range mergeOptions {
		mapOptions[i] = mergeOptions[i]()
	}

	return mapOptions
}
