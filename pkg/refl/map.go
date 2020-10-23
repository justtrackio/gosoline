package refl

import (
	"fmt"
	"reflect"
)

func InterfaceToMapInterfaceInterface(m interface{}) (map[interface{}]interface{}, error) {
	if mii, ok := m.(map[interface{}]interface{}); ok {
		return mii, nil
	}

	mii := make(map[interface{}]interface{})

	v := reflect.ValueOf(m)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Map {
		return mii, fmt.Errorf("value has to be of kind Map but instead is of type %T", m)
	}

	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		mii[k.Interface()] = v.Interface()
	}

	return mii, nil
}

func MapOf(m interface{}) (*Map, error) {
	mapType := reflect.TypeOf(m)
	mapValue := reflect.ValueOf(m)

	if mapType.Kind() == reflect.Ptr {
		mapType = mapType.Elem()
		mapValue = mapValue.Elem()
	}

	if mapType.Kind() != reflect.Map {
		return nil, fmt.Errorf("value has to be of kind Map but instead is of type %T", m)
	}

	keyType := mapType.Key()
	elementType := mapType.Elem()
	elementIsPointer := false

	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem()
		elementIsPointer = true
	}

	return &Map{
		mapType:          mapType,
		mapValue:         mapValue,
		keyType:          keyType,
		elementType:      elementType,
		elementIsPointer: elementIsPointer,
	}, nil
}

type Map struct {
	mapType          reflect.Type
	mapValue         reflect.Value
	keyType          reflect.Type
	elementType      reflect.Type
	elementIsPointer bool
}

func (m *Map) NewElement() interface{} {
	return reflect.New(m.elementType).Interface()
}

func (m *Map) Set(key interface{}, value interface{}) error {
	keyValue := reflect.ValueOf(key)

	if keyValue.Type() != m.keyType {
		return fmt.Errorf("provided key should be of type %v but instead is %v", m.keyType, keyValue.Type())
	}

	valueValue := reflect.ValueOf(value)

	if !m.elementIsPointer {
		valueValue = reflect.Indirect(valueValue)
	}

	m.mapValue.SetMapIndex(keyValue, valueValue)

	return nil
}
