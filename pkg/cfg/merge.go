package cfg

import (
	"fmt"
	"github.com/imdario/mergo"
	"reflect"
)

func Merge(target interface{}, source interface{}) error {
	tt := reflect.TypeOf(target)

	if tt.Kind() != reflect.Ptr {
		return fmt.Errorf("target value has to be a pointer")
	}

	return mergo.Merge(target, source, mergo.WithOverride, mergo.WithAppendSlice)
}
