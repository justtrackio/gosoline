package mdl

type Transformer func(in interface{}) (out interface{})
type TransformerResolver func(view string, version int, in interface{}) (out interface{})
type TransformerMap map[string]map[int]Transformer

func Transform(transformers TransformerMap) TransformerResolver {
	return func(view string, version int, in interface{}) (out interface{}) {
		return transformers[view][version](in)
	}
}
