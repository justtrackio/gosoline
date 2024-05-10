package mdl

type (
	Transformer         func(in any) (out any)
	TransformerResolver func(view string, version int, in any) (out any)
	TransformerMap      map[string]map[int]Transformer
)

func Transform(transformers TransformerMap) TransformerResolver {
	return func(view string, version int, in any) (out any) {
		return transformers[view][version](in)
	}
}
