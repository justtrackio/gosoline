package refl

import (
	"fmt"
	"reflect"
)

func InterfaceToInterfaceSlice(in interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(in)

	if val.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input is not an slice but instead of type %T", in)
	}

	out := make([]interface{}, val.Len())

	for i := 0; i < val.Len(); i++ {
		out[i] = val.Index(i).Interface()
	}

	return out, nil
}

func SliceOf(slice interface{}) (*Slice, error) {
	sliceType := reflect.TypeOf(slice)

	if sliceType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("the slice has to be addressable")
	}

	sliceType = sliceType.Elem()
	sliceValue := reflect.ValueOf(slice)

	if sliceValue.Kind() == reflect.Ptr {
		sliceValue = sliceValue.Elem()
	}

	elementType := sliceType.Elem()
	elementPtr := false

	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem()
		elementPtr = true
	}

	sr := &Slice{
		slice:       slice,
		sliceType:   sliceType,
		sliceValue:  sliceValue,
		elementType: elementType,
		elementPtr:  elementPtr,
	}

	return sr, nil
}

type Slice struct {
	slice       interface{}
	sliceType   reflect.Type
	sliceValue  reflect.Value
	elementType reflect.Type
	elementPtr  bool
}

func (s *Slice) NewElement() interface{} {
	return reflect.New(s.elementType).Interface()
}

func (s *Slice) Append(elem interface{}) error {
	ev := reflect.ValueOf(elem)

	if s.elementPtr == true && ev.Kind() != reflect.Ptr {
		return fmt.Errorf("the value which you try to append to the slice has to be addressable")
	}

	if s.elementPtr == false && ev.Kind() == reflect.Ptr {
		ev = reflect.Indirect(ev)
	}

	s.sliceValue.Set(reflect.Append(s.sliceValue, ev))

	return nil
}
