package cfg

import "github.com/imdario/mergo"

type MergeOption func(mergoConfigs *[]func(*mergo.Config))

func MergeWithOverride(mergoConfigs *[]func(*mergo.Config)) {
	*mergoConfigs = append(*mergoConfigs, mergo.WithOverride)
}

func MergeWithoutOverride(mergoConfigs *[]func(*mergo.Config)) {}
