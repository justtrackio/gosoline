package cfg

import "github.com/spf13/cast"

func sanitize(settings interface{}) (interface{}, error) {
	switch val := settings.(type) {
	case map[string]interface{}:
		return sanitizeMsi(val)

	case []interface{}:
		return sanitizeSlice(val)

	default:
		return cast.ToStringE(val)
	}
}

func sanitizeSlice(slice []interface{}) ([]interface{}, error) {
	for i := 0; i < len(slice); i++ {
		val, err := sanitize(slice[i])

		if err != nil {
			return nil, err
		}

		slice[i] = val
	}

	return slice, nil
}

func sanitizeMsi(settings map[string]interface{}) (map[string]interface{}, error) {
	var err error
	var san interface{}

	for k, v := range settings {
		switch val := v.(type) {
		case map[string]interface{}:
			san, err = sanitizeMsi(val)

		case []interface{}:
			san, err = sanitizeSlice(val)

		default:
			san, err = cast.ToStringE(v)
		}

		if err != nil {
			return nil, err
		}

		settings[k] = san
	}

	return settings, nil
}
