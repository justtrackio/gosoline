package fixtures

import (
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

type NamedFixture struct {
	Name  string
	Value interface{}
}

type NamedFixtureSet []*NamedFixture

func (l *NamedFixtureSet) All() []interface{} {
	values := make([]interface{}, 0)

	for _, named := range *l {
		values = append(values, named.Value)
	}

	return values
}

func (l *NamedFixtureSet) Len() int {
	return len(*l)
}

func (l *NamedFixtureSet) CountIf(f func(elem interface{}) bool) int {
	count := 0

	for _, elem := range *l {
		if f(elem.Value) {
			count++
		}
	}

	return count
}

func (l *NamedFixtureSet) FindFirst(f func(elem interface{}) bool) (interface{}, bool) {
	for _, elem := range *l {
		if f(elem.Value) {
			return elem.Value, true
		}
	}

	return nil, false
}

func (l *NamedFixtureSet) FindAll(f func(elem interface{}) bool) []interface{} {
	a := make([]interface{}, 0)
	for _, elem := range *l {
		if f(elem.Value) {
			a = append(a, elem.Value)
		}
	}

	return a
}

func (l *NamedFixtureSet) GetValueByName(name string) interface{} {
	fixture, ok := funk.FindFirstFunc(*l, func(item *NamedFixture) bool {
		return item.Name == name
	})
	if !ok {
		panic(fmt.Errorf("failed to get value by name"))
	}

	return fixture.Value
}

func (l *NamedFixtureSet) GetValueById(id interface{}) interface{} {
	fixture, ok := funk.FindFirstFunc(*l, func(item *NamedFixture) bool {
		valueId, ok := GetValueId(item.Value)

		return ok && id == valueId
	})

	if !ok {
		panic(fmt.Errorf("failed to get value by id"))
	}

	return fixture.Value
}

func GetValueId(value interface{}) (interface{}, bool) {
	if kvValue, ok := value.(KvStoreFixture); ok {
		return GetValueId(kvValue.Value)
	}

	if kvValue, ok := value.(*KvStoreFixture); ok && kvValue != nil {
		return GetValueId(kvValue.Value)
	}

	if identifiable, ok := value.(mdl.Identifiable); ok {
		return mdl.EmptyIfNil(identifiable.GetId()), true
	}

	model, ok := value.(mdlsub.Model)
	if !ok {
		return nil, false
	}

	modelId := model.GetId()
	rf := reflect.ValueOf(modelId)

	if rf.Kind() != reflect.Ptr {
		return modelId, true
	}

	if mdl.IsNil(modelId) {
		return reflect.New(rf.Type().Elem()), true
	}

	return rf.Elem(), true
}

func GetNamedFixtures(container interface{}) []NamedFixtureSet {
	v := reflect.ValueOf(container)
	for v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("expected a struct when getting named fixtureSets out of %T", container))
	}

	// extract the values of all fields which are a NamedFixtureSet
	result := make([]NamedFixtureSet, 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).CanInterface() {
			continue
		}

		fieldVal := v.Field(i).Interface()

		if val, ok := fieldVal.(NamedFixtureSet); ok {
			result = append(result, val)
		}
	}

	return result
}
