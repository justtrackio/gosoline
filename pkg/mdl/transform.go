package mdl

type (
	Transformer         func(in interface{}) (out interface{})
	TransformerResolver func(view string, version int, in interface{}) (out interface{})
	TransformerMap      map[string]map[int]Transformer
)

func Transform(transformers TransformerMap) TransformerResolver {
	return func(view string, version int, in interface{}) (out interface{}) {
		return transformers[view][version](in)
	}
}
