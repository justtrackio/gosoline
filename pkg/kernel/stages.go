package kernel

import "sort"

type stages map[int]*stage

func (s stages) hasModules() bool {
	// no need to iterate in order as we are only checking
	for _, stage := range s {
		if len(stage.modules.modules) > 0 {
			return true
		}
	}

	return false
}

func (s stages) countForegroundModules() int32 {
	count := int32(0)

	// no need to iterate in order as we are only counting
	for _, stage := range s {
		for _, m := range stage.modules.modules {
			if !m.config.background {
				count++
			}
		}
	}

	return count
}

func (s stages) getIndices() []int {
	keys := make([]int, len(s))
	i := 0

	for k := range s {
		keys[i] = k
		i++
	}

	sort.Ints(keys)

	return keys
}
