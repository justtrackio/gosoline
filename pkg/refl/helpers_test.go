package refl_test

import (
	"github.com/applike/gosoline/pkg/refl"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type testStruct struct{}

func TestIsPointerToSlice(t *testing.T) {
	var nilSlice []string
	var interfacedNilSlice = interface{}(nilSlice)
	var interfacedSlice = interface{}([]string{"abc"})

	tests := map[string]struct {
		Input    interface{}
		Expected bool
	}{
		"nil": {
			Input:    nil,
			Expected: false,
		},
		"int": {
			Input:    0,
			Expected: false,
		},
		"struct": {
			Input:    testStruct{},
			Expected: false,
		},
		"slice": {
			Input:    []string{"abc"},
			Expected: false,
		},
		"nil_ptr_slice": {
			Input:    &nilSlice,
			Expected: true,
		},
		"ptr_slice": {
			Input:    &[]string{"abc"},
			Expected: true,
		},
		"ptr_interfaced_slice": {
			Input:    &interfacedSlice,
			Expected: true,
		},
		"ptr_interfaced_nil_slice": {
			Input:    &interfacedNilSlice,
			Expected: true,
		},
		"ptr_slice_interfaces": {
			Input:    &[]interface{}{"abc"},
			Expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := refl.IsPointerToSlice(tt.Input)

			assert.Equal(t, tt.Expected, result, name)
		})
	}
}

func TestIsPointerToStruct(t *testing.T) {
	var nilStruct testStruct
	var interfacedStruct = interface{}(testStruct{})

	tests := map[string]struct {
		Input    interface{}
		Expected bool
	}{
		"nil": {
			Input:    nil,
			Expected: false,
		},
		"int": {
			Input:    0,
			Expected: false,
		},
		"struct": {
			Input:    testStruct{},
			Expected: false,
		},
		"slice": {
			Input:    []string{"abc"},
			Expected: false,
		},
		"ptr_nil_struct": {
			Input:    &nilStruct,
			Expected: true,
		},
		"ptr_struct": {
			Input:    &testStruct{},
			Expected: true,
		},
		"ptr_interfaced_struct": {
			Input:    &interfacedStruct,
			Expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := refl.IsPointerToStruct(tt.Input)

			assert.Equal(t, tt.Expected, result, name)
		})
	}
}

func TestFindBaseType(t *testing.T) {
	var nilStruct testStruct
	var interfacedStruct = interface{}(testStruct{})
	var interfacedSlice = interface{}([]interface{}{"abc"})

	tests := map[string]struct {
		Input    interface{}
		Expected reflect.Kind
	}{
		"int": {
			Input:    0,
			Expected: reflect.Int,
		},
		"struct": {
			Input:    testStruct{},
			Expected: reflect.Struct,
		},
		"ptr_nil_struct": {
			Input:    &nilStruct,
			Expected: reflect.Struct,
		},
		"ptr_struct": {
			Input:    &testStruct{},
			Expected: reflect.Struct,
		},
		"ptr_interfaced_struct": {
			Input:    &interfacedStruct,
			Expected: reflect.Struct,
		},
		"slice": {
			Input:    []string{"abc"},
			Expected: reflect.String,
		},
		"ptr_slice": {
			Input:    &[]string{"abc"},
			Expected: reflect.String,
		},
		"ptr_slice_interfaces": {
			Input:    &[]interface{}{"abc"},
			Expected: reflect.String,
		},
		"ptr_interfaced_slice_interfaces": {
			Input:    &interfacedSlice,
			Expected: reflect.String,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, _ := refl.ResolveBaseTypeAndValue(tt.Input)

			assert.Equal(t, tt.Expected, result.Kind(), name)
		})
	}

	result, _ := refl.ResolveBaseTypeAndValue(nil)
	assert.Equal(t, nil, result, "nil")
}

func TestGetTypedValue(t *testing.T) {
	var interfacedStruct = interface{}(testStruct{})
	var interfacedSlice = interface{}([]string{})

	tests := map[string]struct {
		Input    interface{}
		Expected reflect.Value
	}{
		"int": {
			Input:    0,
			Expected: reflect.ValueOf(0),
		},
		"ptr_struct": {
			Input:    &testStruct{},
			Expected: reflect.ValueOf(testStruct{}),
		},
		"ptr_interfaced_struct": {
			Input:    &interfacedStruct,
			Expected: reflect.ValueOf(testStruct{}),
		},
		"ptr_slice": {
			Input:    &[]string{"abc"},
			Expected: reflect.ValueOf([]string{}),
		},
		"ptr_interfaced_slice": {
			Input:    &interfacedSlice,
			Expected: reflect.ValueOf([]string{}),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := refl.GetTypedValue(tt.Input)

			assert.Equal(t, tt.Expected.Kind(), result.Kind(), name)
		})
	}
}

func TestCreatePointerToSliceOfTypeAndSize(t *testing.T) {
	input := interface{}([]interface{}{""})
	result := refl.CreatePointerToSliceOfTypeAndSize(&input, 10)

	casted, castable := result.(*[]string)

	assert.True(t, castable)
	assert.Len(t, *casted, 10)
}

func TestCopyPointerSlice(t *testing.T) {
	target := make([]string, 0)
	source := []string{"abc", "def"}

	refl.CopyPointerSlice(&target, &source)

	assert.Equal(t, source, target)
}
