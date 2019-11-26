package cfg

import (
	"github.com/spf13/cast"
	"reflect"
	"time"
)

func StringToTimeHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(time.Time{}) {
		return data, nil
	}

	return cast.ToTimeE(data)
}
