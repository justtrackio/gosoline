package mdl

const NamedView = "named"

type NamedOutputV0 struct {
	Id   *uint   `json:"id"`
	Name *string `json:"name"`
}

type Nameable interface {
	Identifiable[uint]
	GetName() *string
}

func NamedOutput(in any) any {
	if IsNil(in) {
		return &NamedOutputV0{}
	}

	rm := in.(Nameable)

	return &NamedOutputV0{
		Id:   rm.GetId(),
		Name: rm.GetName(),
	}
}
