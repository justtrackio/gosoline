package cfg

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const PathSeparator = "."
const arrayAccessRegexString = `^(.+)\[([0-9]+)\]$`

var arrayAccessRegex = regexp.MustCompile(arrayAccessRegexString)

type Map map[string]interface{}

func NewMap() Map {
	return make(map[string]interface{})
}

func (m Map) Msi() map[string]interface{} {
	return m
}

func (m Map) Get(selector string) interface{} {
	return m.access(m, selector, nil, false)
}

func (m Map) Has(selector string) bool {
	if m == nil {
		return false
	}
	return m.Get(selector) != nil
}

func (m Map) Set(key string, value interface{}) {
	m.access(m, key, value, true)
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

// access accesses the object using the selector and performs the
// appropriate action.
func (m Map) access(current interface{}, selector string, value interface{}, isSet bool) interface{} {
	selSegs := strings.SplitN(selector, PathSeparator, 2)
	thisSel := selSegs[0]
	index := -1

	if strings.Contains(thisSel, "[") {
		index, thisSel = getIndex(thisSel)
	}

	if curMap, ok := current.(Map); ok {
		current = map[string]interface{}(curMap)
	}
	// get the object in question
	switch current.(type) {
	case map[string]interface{}:
		curMSI := current.(map[string]interface{})
		if len(selSegs) <= 1 && isSet {
			m.doSet(curMSI, thisSel, index, value)
			return nil
		}

		_, ok := curMSI[thisSel].(map[string]interface{})
		if (curMSI[thisSel] == nil || !ok) && index == -1 && isSet {
			curMSI[thisSel] = map[string]interface{}{}
		}

		// create new array if missing
		if curMSI[thisSel] == nil && isSet && index > -1 {
			// type of interface{}
			at := reflect.TypeOf((*interface{})(nil)).Elem()
			st := reflect.SliceOf(at)
			sv := reflect.MakeSlice(st, 0, 4)

			array := sv.Interface().([]interface{})
			if index >= len(array) && isSet {
				m.fillSlice(&array, index, len(selSegs), value)
			}

			curMSI[thisSel] = array
		}

		// expand existing array
		if array, ok := curMSI[thisSel].([]interface{}); ok && isSet && index > -1 && index >= len(array) {
			m.fillSlice(&array, index, len(selSegs), value)
			curMSI[thisSel] = array
		}

		current = curMSI[thisSel]
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
		current = m.access(current, selSegs[1], value, isSet)
	}

	return current
}

func (m Map) doSet(current map[string]interface{}, key string, index int, value interface{}) {
	if index < 0 {
		current[key] = value
		return
	}

	if _, ok := current[key]; !ok {
		vt := reflect.TypeOf(value)
		vv := reflect.ValueOf(value)
		st := reflect.SliceOf(vt)

		array := reflect.MakeSlice(st, 0, 4)
		array = reflect.Append(array, vv)

		current[key] = array.Interface()
		return
	}

	array := current[key]

	vv := reflect.ValueOf(value)
	sv := reflect.ValueOf(array)

	if index < sv.Len() {
		sv.Index(index).Set(vv)
		return
	}

	for i := sv.Len(); i <= index; i++ {
		sv = reflect.Append(sv, reflect.Zero(vv.Type()))
	}

	sv.Index(index).Set(vv)
	current[key] = sv.Interface()
	return
}

func (m Map) fillSlice(array *[]interface{}, index int, segmentCount int, value interface{}) {
	va := reflect.ValueOf(array).Elem()
	vv := reflect.ValueOf(value)

	if segmentCount > 1 {
		vv = reflect.ValueOf(map[string]interface{}{})
	} else {
		vv = reflect.Zero(vv.Type())
	}

	for i := va.Len(); i <= index; i++ {
		var nv reflect.Value

		if segmentCount > 1 {
			nv = reflect.ValueOf(map[string]interface{}{})
		} else {
			nv = reflect.Zero(vv.Type())
		}

		va.Set(reflect.Append(va, nv))
	}
}
