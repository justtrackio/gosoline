package mapx

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	PathSeparator          = "."
	arrayAccessRegexString = `^(.+)\[([0-9]+)\]$`
)

type Msier interface {
	Msi() map[string]interface{}
}

var arrayAccessRegex = regexp.MustCompile(arrayAccessRegexString)

type MapX struct {
	lck sync.Mutex
	msn map[string]*MapXNode
}

func NewMapX(msis ...map[string]interface{}) *MapX {
	m := &MapX{
		msn: make(map[string]*MapXNode),
	}

	if len(msis) > 0 {
		m.msn = msiToMsn(msis[0])
	}

	return m
}

func (m *MapX) Msi() map[string]interface{} {
	return nodeMsnToMsi(m.msn)
}

func (m *MapX) Keys() []string {
	keys := make([]string, 0, len(m.msn))

	for k := range m.msn {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func (m *MapX) Get(selector string) *MapXNode {
	m.lck.Lock()
	defer m.lck.Unlock()

	val := m.access(m.msn, selector, nil, &OpMode{})

	return &MapXNode{value: val}
}

func (m *MapX) Has(selector string) bool {
	m.lck.Lock()

	if len(m.msn) == 0 {
		m.lck.Unlock()
		return false
	}

	m.lck.Unlock()

	return m.Get(selector).value != nil
}

func (m *MapX) Set(key string, value interface{}, options ...MapOption) {
	m.lck.Lock()
	defer m.lck.Unlock()

	mode := &OpMode{
		IsSet: true,
	}

	for _, opt := range options {
		opt(mode)
	}

	value = m.prepareInput(value)
	m.access(m.msn, key, value, mode)
}

func (m *MapX) prepareInput(value interface{}) interface{} {
	switch t := value.(type) {
	case Msier:
		msi := t.Msi()
		value = msiToMsn(msi)

	case map[string]interface{}:
		value = msiToMsn(t)

	case []interface{}:
		cpy := make([]interface{}, len(t))

		for i, val := range t {
			cpy[i] = interfaceToMapNode(val).value
		}

		value = cpy
	}

	return value
}

// access accesses the object using the selector and performs the
// appropriate action.
func (m *MapX) access(current interface{}, selector string, value interface{}, mode *OpMode) interface{} {
	selector = strings.Trim(selector, ".")
	selSegs := strings.SplitN(selector, PathSeparator, 2)

	thisSel := selSegs[0]
	index := -1

	if strings.Contains(thisSel, "[") {
		index, thisSel = getIndex(thisSel)
	}

	// get the object in question
	switch current.(type) {
	case map[string]*MapXNode:
		curMsn := current.(map[string]*MapXNode)

		if len(selSegs) <= 1 && mode.IsSet {
			m.doSet(curMsn, thisSel, index, value, mode)
			return nil
		}

		// check if items exist on get
		if curMsn[thisSel] == nil && !mode.IsSet {
			return nil
		}

		if curMsn[thisSel] == nil && index == -1 && mode.IsSet {
			curMsn[thisSel] = &MapXNode{value: make(map[string]*MapXNode)}
		}

		// create new array if missing
		if curMsn[thisSel] == nil && mode.IsSet && index > -1 {
			// type of interface{}
			at := reflect.TypeOf((*interface{})(nil)).Elem()
			st := reflect.SliceOf(at)
			sv := reflect.MakeSlice(st, 0, 4)

			array := sv.Interface().([]interface{})
			if index >= len(array) && mode.IsSet {
				m.fillSlice(&array, index, len(selSegs), value)
			}

			curMsn[thisSel] = &MapXNode{value: array}
		}

		// expand existing array
		if array, ok := curMsn[thisSel].value.([]interface{}); ok && mode.IsSet && index > -1 && index >= len(array) {
			m.fillSlice(&array, index, len(selSegs), value)
			curMsn[thisSel] = &MapXNode{value: array}
		}

		// initialize empty value
		if curMsn[thisSel].value == nil && len(selSegs) > 1 {
			curMsn[thisSel].value = map[string]*MapXNode{}
		}

		current = curMsn[thisSel].value
	default:
		current = nil
	}

	// do we need to access the item of an array?
	if index > -1 {
		if array, ok := current.([]interface{}); ok {
			if index < len(array) {
				current = array[index]
			} else {
				current = nil
			}
		}
	}

	if len(selSegs) > 1 {
		current = m.access(current, selSegs[1], value, mode)
	}

	return current
}

func (m *MapX) doSet(current map[string]*MapXNode, key string, index int, value interface{}, mode *OpMode) {
	reflectValue := reflect.ValueOf(value)

	if index < 0 && reflectValue.Kind() == reflect.Slice {
		if _, ok := current[key]; ok && mode.SkipExisting {
			return
		}

		m.doSetSlice(current, key, reflectValue)
		return
	}

	if index < 0 {
		if _, ok := current[key]; ok && mode.SkipExisting {
			return
		}

		current[key] = interfaceToMapNode(value)
		return
	}

	if _, ok := current[key]; !ok {
		array := make([]interface{}, index+1)
		array[index] = value

		current[key] = &MapXNode{value: array}
		return
	}

	array := current[key].value
	arrayValue := reflect.ValueOf(array)

	if index < arrayValue.Len() {
		if mode.SkipExisting {
			return
		}

		arrayValue.Index(index).Set(reflectValue)
		return
	}

	for i := arrayValue.Len(); i <= index; i++ {
		arrayValue = reflect.Append(arrayValue, reflect.Zero(reflectValue.Type()))
	}

	arrayValue.Index(index).Set(reflectValue)
	current[key] = &MapXNode{value: arrayValue.Interface()}
}

func (m *MapX) doSetSlice(current map[string]*MapXNode, key string, value reflect.Value) {
	sl := make([]interface{}, value.Len())

	for i := 0; i < value.Len(); i++ {
		sl[i] = value.Index(i).Interface()
	}

	current[key] = &MapXNode{value: sl}
}

func (m *MapX) fillSlice(array *[]interface{}, index int, segmentCount int, value interface{}) {
	va := reflect.ValueOf(array).Elem()
	vv := reflect.ValueOf(value)

	if segmentCount > 1 {
		vv = reflect.ValueOf(map[string]*MapXNode{})
	} else {
		vv = reflect.Zero(vv.Type())
	}

	for i := va.Len(); i <= index; i++ {
		var nv reflect.Value

		if segmentCount > 1 {
			nv = reflect.ValueOf(map[string]*MapXNode{})
		} else {
			nv = reflect.Zero(vv.Type())
		}

		va.Set(reflect.Append(va, nv))
	}
}

func (m *MapX) Merge(key string, source interface{}, options ...MapOption) {
	if msier, ok := source.(Msier); ok {
		source = msier.Msi()
	}

	sourceValue := reflect.ValueOf(source)

	var mapIter *reflect.MapIter
	var elementKey string
	var elementValue interface{}

	if sourceValue.Kind() == reflect.Map {
		if key == "." && sourceValue.Len() == 0 {
			return
		}

		if !m.Has(key) && sourceValue.Len() == 0 {
			m.Set(key, map[string]interface{}{}, options...)
			return
		}

		mapIter = sourceValue.MapRange()
		for mapIter.Next() {
			elementKey = fmt.Sprintf("%s.%s", key, mapIter.Key())
			elementValue = mapIter.Value().Interface()

			m.Merge(elementKey, elementValue, options...)
		}

		return
	}

	if sourceValue.Kind() == reflect.Slice {
		if !m.Has(key) {
			m.Set(key, []interface{}{}, options...)
		}

		for i := 0; i < sourceValue.Len(); i++ {
			elementKey = fmt.Sprintf("%s[%d]", key, i)
			elementValue = sourceValue.Index(i).Interface()

			m.Merge(elementKey, elementValue, options...)
		}

		return
	}

	m.Set(key, source, options...)
}

// getIndex returns the index, which is hold in s by two braches.
// It also returns s withour the index part, e.g. name[1] will return (1, name).
// If no index is found, -1 is returned
func getIndex(s string) (int, string) {
	arrayMatches := arrayAccessRegex.FindStringSubmatch(s)
	if len(arrayMatches) > 0 {
		// Get the key into the map
		selector := arrayMatches[1]
		// Get the index into the array at the key
		// We know this cannt fail because arrayMatches[2] is an int for sure
		index, _ := strconv.Atoi(arrayMatches[2])
		return index, selector
	}
	return -1, s
}
