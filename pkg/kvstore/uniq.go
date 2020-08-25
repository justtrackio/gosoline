package kvstore

import (
	"fmt"
)

func UniqKeys(keys []interface{}) ([]interface{}, error) {
	length := len(keys)
	uniqKeys := make([]interface{}, 0, length)
	seen := make(map[string]bool, length)

	for i := 0; i < length; i++ {
		keyString, err := CastKeyToString(keys[i])

		if err != nil {
			return nil, fmt.Errorf("can not build string key from %T %v: %w", keys[i], keys[i], err)
		}

		if _, ok := seen[keyString]; ok {
			continue
		}

		seen[keyString] = true
		uniqKeys = append(uniqKeys, keys[i])
	}

	return uniqKeys, nil
}
