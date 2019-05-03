package mdl

const NamedView = "named"

type NamedOutputV0 struct {
	Id   *uint   `json:"id"`
	Name *string `json:"name"`
}

type Nameable interface {
	Identifiable
	GetName() *string
}

func NamedOutput(in interface{}) interface{} {
	rm := in.(Nameable)

	return &NamedOutputV0{
		Id:   rm.GetId(),
		Name: rm.GetName(),
	}
}
