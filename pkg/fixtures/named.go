package fixtures

import (
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

type NamedFixture[T any] struct {
	Name  string
	Value T
}

type NamedFixtures[T any] []*NamedFixture[T]

// All specifically returns a []any instead of a []T, so the fixture loader code doesn't complain that it can't cast a []T to []any
func (l *NamedFixtures[T]) All() []any {
	values := make([]any, 0)

	for _, named := range *l {
		values = append(values, named.Value)
	}

	return values
}

func (l *NamedFixtures[T]) Len() int {
	return len(*l)
}

func (l *NamedFixtures[T]) CountIf(f func(elem T) bool) int {
	count := 0

	for _, elem := range *l {
		if f(elem.Value) {
			count++
		}
	}

	return count
}

func (l *NamedFixtures[T]) FindFirst(f func(elem T) bool) (T, bool) {
	for _, elem := range *l {
		if f(elem.Value) {
			return elem.Value, true
		}
	}

	t := new(T)

	return *(t), false
}

func (l *NamedFixtures[T]) FindAll(f func(elem T) bool) []T {
	a := make([]T, 0)
	for _, elem := range *l {
		if f(elem.Value) {
			a = append(a, elem.Value)
		}
	}

	return a
}

func (l *NamedFixtures[T]) GetValueByName(name string) T {
	fixture, ok := funk.FindFirstFunc(*l, func(item *NamedFixture[T]) bool {
		return item.Name == name
	})
	if !ok {
		panic(fmt.Errorf("failed to get value by name: %s", name))
	}

	return fixture.Value
}

func (l *NamedFixtures[T]) GetValueById(id any) T {
	if l.Len() == 0 {
		panic(fmt.Errorf("can not find id = %v in empty fixture set", id))
	}

	fixture, ok := funk.FindFirstFunc(*l, func(item *NamedFixture[T]) bool {
		valueId, ok := GetValueId(item.Value)

		return ok && id == valueId
	})

	if !ok {
		panic(fmt.Errorf("failed to get value by id = %v, type = %T", id, (*l)[0].Value))
	}

	return fixture.Value
}

func GetValueId(value any) (any, bool) {
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

func GetNamedFixtures(container any) []NamedFixtures[any] {
	v := reflect.ValueOf(container)
	for v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("expected a struct when getting named fixtureSets out of %T", container))
	}

	// extract the values of all fields which are a NamedFixtures
	result := make([]NamedFixtures[any], 0, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).CanInterface() {
			continue
		}

		fieldVal := v.Field(i).Interface()

		if val, ok := fieldVal.(NamedFixtures[any]); ok {
			result = append(result, val)
		}
	}

	return result
}
