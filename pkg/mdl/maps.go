package mdl

type ValueStringMap map[string]interface{}

func (m ValueStringMap) GetMap(key string) (ValueStringMap, bool) {
	if m == nil {
		return nil, false
	}

	if _, ok := m[key]; !ok {
		return nil, false
	}

	val, ok := m[key].(map[string]interface{})

	return ValueStringMap(val), ok
}

func (m ValueStringMap) GetString(key string) (string, bool) {
	if m == nil {
		return "", false
	}

	if _, ok := m[key]; !ok {
		return "", false
	}

	val, ok := m[key].(string)

	return val, ok
}

type ValueInterfaceMap map[interface{}]interface{}

func (m ValueInterfaceMap) GetMap(key string) (ValueInterfaceMap, bool) {
	if _, ok := m[key]; !ok {
		return nil, false
	}

	val, ok := m[key].(ValueInterfaceMap)

	return val, ok
}

func (m ValueInterfaceMap) GetString(key string) (string, bool) {
	if _, ok := m[key]; !ok {
		return "", false
	}

	val, ok := m[key].(string)

	return val, ok
}

type PointerMap map[interface{}]interface{}

func (m PointerMap) GetString(key interface{}) (*string, bool) {
	if _, ok := m[key]; !ok {
		return nil, false
	}

	if val, ok := m[key].(string); ok {
		return &val, ok
	}

	return nil, false
}
