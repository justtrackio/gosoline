package cfg

import (
	"fmt"
	"reflect"
	"time"
)

type Sanitizer func(in interface{}) (interface{}, error)

func Sanitize(key string, value interface{}, sanitizers []Sanitizer) (interface{}, error) {
	switch val := value.(type) {
	case []interface{}:
		return sanitizeSlice(key, val, sanitizers)

	case map[string]interface{}:
		return sanitizeMap(key, val, sanitizers)

	default:
		return sanitizeValue(key, val, sanitizers)
	}
}

func sanitizeValue(key string, val interface{}, sanitizers []Sanitizer) (interface{}, error) {
	var err error
	var san = val

	for _, sanitizer := range sanitizers {
		if san, err = sanitizer(san); err != nil {
			return nil, fmt.Errorf("can not apply sanitizer on key %s: %w", key, err)
		}
	}

	return san, nil
}

func sanitizeSlice(key string, values []interface{}, sanitizers []Sanitizer) ([]interface{}, error) {
	var err error
	var san interface{}

	for i, val := range values {
		k := fmt.Sprintf("%s.%d", key, i)

		if san, err = Sanitize(k, val, sanitizers); err != nil {
			return nil, fmt.Errorf("can not sanitize slice element %s of type %T: %w", k, val, err)
		}

		values[i] = san
	}

	return values, nil
}

func sanitizeMap(rootKey string, values map[string]interface{}, sanitizers []Sanitizer) (map[string]interface{}, error) {
	var err error
	var san interface{}

	for key, val := range values {
		k := fmt.Sprintf("%s.%s", rootKey, key)

		if san, err = Sanitize(k, val, sanitizers); err != nil {
			return nil, fmt.Errorf("can not sanitize map element %s of type %T: %w", k, val, err)
		}

		values[key] = san
	}

	return values, nil
}

func TimeSanitizer(in interface{}) (interface{}, error) {
	if reflect.TypeOf(in) != reflect.TypeOf(time.Time{}) {
		return in, nil
	}

	tm := in.(time.Time)

	return tm.Format(time.RFC3339), nil
}
