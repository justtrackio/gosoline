package mapx

import (
	"fmt"

	"github.com/spf13/cast"
)

type MapXNode struct {
	value any
}

func msiToMsn(src map[string]any) map[string]*MapXNode {
	target := make(map[string]*MapXNode)

	for k, v := range src {
		target[k] = interfaceToMapNode(v)
	}

	return target
}

func interfaceToMapNode(val any) *MapXNode {
	switch val := val.(type) {
	case map[string]any:
		msn := msiToMsn(val)
		return &MapXNode{value: msn}

	case []map[string]any:
		slice := make([]map[string]*MapXNode, len(val))

		for i, elem := range val {
			slice[i] = msiToMsn(elem)
		}

		return &MapXNode{value: slice}

	case []any:
		slice := make([]any, len(val))

		for i, v := range val {
			slice[i] = interfaceToMapNode(v).value
		}

		return &MapXNode{value: slice}

	default:
		return &MapXNode{value: val}
	}
}

func nodeMsnToMsi(msn map[string]*MapXNode) map[string]any {
	msi := make(map[string]any)

	for k, node := range msn {
		switch val := node.value.(type) {
		case map[string]*MapXNode:
			subMsi := make(map[string]any)

			for k, node := range val {
				switch val := node.value.(type) {
				case map[string]*MapXNode:
					subMsi[k] = nodeMsnToMsi(val)
				case []any:
					subMsi[k] = nodeSliceToSlice(val)
				default:
					subMsi[k] = val
				}
			}

			msi[k] = subMsi

		case []any:
			msi[k] = nodeSliceToSlice(val)

		default:
			msi[k] = node.value
		}
	}

	return msi
}

func nodeSliceToSlice(val []any) []any {
	slice := make([]any, len(val))

	for i, elem := range val {
		switch val := elem.(type) {
		case map[string]*MapXNode:
			slice[i] = nodeMsnToMsi(val)
		case []any:
			slice[i] = nodeSliceToSlice(val)
		default:
			slice[i] = val
		}
	}

	return slice
}

func (n *MapXNode) Data() any {
	switch val := n.value.(type) {
	case map[string]*MapXNode:
		return nodeMsnToMsi(val)
	case []any:
		return nodeSliceToSlice(val)
	default:
		return val
	}
}

func (n *MapXNode) IsMap() bool {
	_, ok := n.value.(map[string]*MapXNode)
	return ok
}

func (n *MapXNode) Map() (*MapX, error) {
	if msn, ok := n.value.(map[string]*MapXNode); ok {
		return &MapX{
			msn: msn,
		}, nil
	}

	return nil, fmt.Errorf("value should be of type map[string]*MapXNode but instead is %T", n.value)
}

func (n *MapXNode) Msi() (map[string]any, error) {
	var ok bool
	var msn map[string]*MapXNode

	if msn, ok = n.value.(map[string]*MapXNode); !ok {
		return nil, fmt.Errorf("value should be of type map[string]*MapXNode but instead is %T", n.value)
	}

	msi := nodeMsnToMsi(msn)

	return msi, nil
}

func (n *MapXNode) Slice() ([]any, error) {
	var ok bool
	var slice []any

	if slice, ok = n.value.([]any); !ok {
		return nil, fmt.Errorf("value should be of type []any but instead is %T", n.value)
	}

	return nodeSliceToSlice(slice), nil
}

func (n *MapXNode) StringSlice() ([]string, error) {
	return cast.ToStringSliceE(n.value)
}
