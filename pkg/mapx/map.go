package mapx

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/justtrackio/gosoline/pkg/funk"
)

const (
	PathSeparator          = "."
	arrayAccessRegexString = `^(.+)\[(\d+)\]$`
)

type Msier interface {
	Msi() map[string]any
}

var arrayAccessRegex = regexp.MustCompile(arrayAccessRegexString)

type MapX struct {
	lck sync.Mutex
	msn map[string]*MapXNode
}

func NewMapX(msis ...map[string]any) *MapX {
	m := &MapX{
		msn: make(map[string]*MapXNode),
	}

	if len(msis) > 0 {
		m.msn = msiToMsn(msis[0])
	}

	return m
}

func (m *MapX) Msi() map[string]any {
	m.lck.Lock()
	defer m.lck.Unlock()

	dc := copyVal(m.msn).(map[string]*MapXNode)

	return nodeMsnToMsi(dc)
}

func (m *MapX) Keys() []string {
	keys := make([]string, 0, len(m.msn))

	for k := range m.msn {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func (m *MapX) Get(key string) *MapXNode {
	m.lck.Lock()
	defer m.lck.Unlock()

	val := m.doGet(key).value

	return &MapXNode{
		value: copyVal(val),
	}
}

func (m *MapX) doGet(key string) *MapXNode {
	if key == "." {
		return &MapXNode{value: m.msn}
	}

	val := m.access(m.msn, key, nil, &OpMode{})

	return &MapXNode{value: val}
}

func (m *MapX) Has(key string) bool {
	m.lck.Lock()
	defer m.lck.Unlock()

	return m.doesHave(key)
}

func (m *MapX) doesHave(key string) bool {
	if len(m.msn) == 0 {
		return false
	}

	return m.doGet(key).value != nil
}

func (m *MapX) Append(key string, values ...any) error {
	m.lck.Lock()
	defer m.lck.Unlock()

	if !m.doesHave(key) {
		m.doSet(key, values)

		return nil
	}

	var err error
	var slice []any

	if slice, err = m.doGet(key).Slice(); err != nil {
		return fmt.Errorf("current value is not a slice: %w", err)
	}

	slice = append(slice, values...)
	m.doSet(key, slice)

	return nil
}

func (m *MapX) Set(key string, value any, options ...MapOption) {
	m.lck.Lock()
	defer m.lck.Unlock()

	m.doSet(key, value, options...)
}

func (m *MapX) doSet(key string, value any, options ...MapOption) {
	mode := &OpMode{
		IsSet: true,
	}

	for _, opt := range options {
		opt(mode)
	}

	value = m.prepareInput(value)
	m.access(m.msn, key, value, mode)
}

func (m *MapX) prepareInput(value any) any {
	switch t := value.(type) {
	case Msier:
		msi := t.Msi()
		value = msiToMsn(msi)

	case map[string]any:
		value = msiToMsn(t)

	case []any:
		cpy := make([]any, len(t))

		for i, val := range t {
			cpy[i] = interfaceToMapNode(val).value
		}

		value = cpy
	}

	return value
}

// access accesses the object using the selector and performs the
// appropriate action.
//

func (m *MapX) access(current any, selector string, value any, mode *OpMode) any {
	selector = strings.Trim(selector, ".")
	selSegs := SplitUnescapedDotN(selector, 2)

	if len(selSegs) > 1 {
		selSegs = []string{selSegs[0], strings.Join(selSegs[1:], ".")}
	}

	thisSel := selSegs[0]
	index := -1

	if strings.Contains(thisSel, "[") {
		index, thisSel = getIndex(thisSel)
	}

	// get the object in question
	switch curMsn := current.(type) {
	case map[string]*MapXNode:
		if current = m.accessMap(curMsn, selSegs, thisSel, index, value, mode); current == nil {
			return nil
		}
	default:
		current = nil
	}

	// do we need to access the item of an array?
	if array, ok := current.([]any); ok && index > -1 {
		if index < len(array) {
			current = array[index]
		} else {
			current = nil
		}
	}

	if len(selSegs) > 1 {
		current = m.access(current, selSegs[1], value, mode)
	}

	return current
}

func (m *MapX) accessMap(curMsn map[string]*MapXNode, selSegs []string, thisSel string, index int, value any, mode *OpMode) any {
	if len(selSegs) <= 1 && mode.IsSet {
		m.apply(curMsn, thisSel, index, value, mode)

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
		// type of any
		at := reflect.TypeOf((*any)(nil)).Elem()
		st := reflect.SliceOf(at)
		sv := reflect.MakeSlice(st, 0, 4)

		array := sv.Interface().([]any)
		if index >= len(array) && mode.IsSet {
			m.fillSlice(&array, index, len(selSegs), value)
		}

		curMsn[thisSel] = &MapXNode{value: array}
	}

	// expand existing array
	if array, ok := curMsn[thisSel].value.([]any); ok && mode.IsSet && index > -1 && index >= len(array) {
		m.fillSlice(&array, index, len(selSegs), value)
		curMsn[thisSel] = &MapXNode{value: array}
	}

	// initialize empty value
	if curMsn[thisSel].value == nil && len(selSegs) > 1 {
		curMsn[thisSel].value = map[string]*MapXNode{}
	}

	return curMsn[thisSel].value
}

func (m *MapX) apply(current map[string]*MapXNode, key string, index int, value any, mode *OpMode) {
	reflectValue := reflect.ValueOf(value)

	if index < 0 && reflectValue.Kind() == reflect.Slice {
		if _, ok := current[key]; ok && mode.SkipExisting {
			return
		}

		m.applySlice(current, key, reflectValue)

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
		array := make([]any, index+1)
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

func (m *MapX) applySlice(current map[string]*MapXNode, key string, value reflect.Value) {
	sl := make([]any, value.Len())

	for i := 0; i < value.Len(); i++ {
		sl[i] = value.Index(i).Interface()
	}

	current[key] = &MapXNode{value: sl}
}

func (m *MapX) fillSlice(array *[]any, index int, segmentCount int, value any) {
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

func (m *MapX) Merge(key string, source any, options ...MapOption) {
	m.lck.Lock()
	defer m.lck.Unlock()

	m.doMerge(key, source, options...)
}

func (m *MapX) doMerge(key string, source any, options ...MapOption) {
	if msier, ok := source.(Msier); ok {
		source = msier.Msi()
	}

	sourceValue := reflect.ValueOf(source)

	var mapIter *reflect.MapIter
	var elementKey string
	var elementValue any

	if sourceValue.Kind() == reflect.Map {
		if key == "." && sourceValue.Len() == 0 {
			return
		}

		if !m.doesHave(key) && sourceValue.Len() == 0 {
			m.doSet(key, map[string]any{}, options...)

			return
		}

		mapIter = sourceValue.MapRange()
		for mapIter.Next() {
			elementKey = fmt.Sprintf("%s.%v", key, mapIter.Key())
			elementValue = mapIter.Value().Interface()

			m.doMerge(elementKey, elementValue, options...)
		}

		return
	}

	if sourceValue.Kind() == reflect.Slice {
		if !m.doesHave(key) {
			m.doSet(key, []any{}, options...)
		}

		for i := 0; i < sourceValue.Len(); i++ {
			elementKey = fmt.Sprintf("%s[%d]", key, i)
			elementValue = sourceValue.Index(i).Interface()

			m.doMerge(elementKey, elementValue, options...)
		}

		return
	}

	m.doSet(key, source, options...)
}

func (m *MapX) String() string {
	return fmt.Sprint(m.Msi())
}

func copyVal(val any) any {
	switch current := val.(type) {
	case map[string]*MapXNode:
		res := make(map[string]*MapXNode, len(current))
		for k, v := range current {
			res[k] = copyVal(v).(*MapXNode)
		}

		return res
	case *MapXNode:
		return &MapXNode{value: copyVal(current.value)}
	case []any:
		return funk.Map(current, copyVal)
	default:
		return val
	}
}

// getIndex returns the index, which is hold in s by two branches.
// It also returns s without the index part, e.g. name[1] will return (1, name).
// If no index is found, -1 is returned
func getIndex(s string) (index int, selector string) {
	arrayMatches := arrayAccessRegex.FindStringSubmatch(s)

	if len(arrayMatches) > 0 {
		// Get the key into the map
		selector = arrayMatches[1]
		// Get the index into the array at the key
		//nolint:errcheck // We know this cannot fail because arrayMatches[2] is an int for sure
		index, _ = strconv.Atoi(arrayMatches[2])

		return index, selector
	}

	return -1, s
}
