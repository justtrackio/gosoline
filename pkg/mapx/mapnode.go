package mapx

import "fmt"

type MapXNode struct {
	value interface{}
}

func msiToMsn(src map[string]interface{}) map[string]*MapXNode {
	target := make(map[string]*MapXNode)

	for k, v := range src {
		target[k] = interfaceToMapNode(v)
	}

	return target
}

func interfaceToMapNode(val interface{}) *MapXNode {
	switch val := val.(type) {
	case map[string]interface{}:
		msn := msiToMsn(val)
		return &MapXNode{value: msn}

	case []map[string]interface{}:
		slice := make([]map[string]*MapXNode, len(val))

		for i, elem := range val {
			slice[i] = msiToMsn(elem)
		}

		return &MapXNode{value: slice}

	case []interface{}:
		slice := make([]interface{}, len(val))

		for i, v := range val {
			slice[i] = interfaceToMapNode(v).value
		}

		return &MapXNode{value: slice}

	default:
		return &MapXNode{value: val}
	}
}

func nodeMsnToMsi(msn map[string]*MapXNode) map[string]interface{} {
	msi := make(map[string]interface{})

	for k, node := range msn {
		switch val := node.value.(type) {
		case map[string]*MapXNode:
			subMsi := make(map[string]interface{})

			for k, node := range val {
				switch val := node.value.(type) {
				case map[string]*MapXNode:
					subMsi[k] = nodeMsnToMsi(val)
				case []interface{}:
					subMsi[k] = nodeSliceToSlice(val)
				default:
					subMsi[k] = val
				}
			}

			msi[k] = subMsi

		case []interface{}:
			msi[k] = nodeSliceToSlice(val)

		default:
			msi[k] = node.value
		}
	}

	return msi
}

func nodeSliceToSlice(val []interface{}) []interface{} {
	slice := make([]interface{}, len(val))

	for i, elem := range val {
		switch val := elem.(type) {
		case map[string]*MapXNode:
			slice[i] = nodeMsnToMsi(val)
		case []interface{}:
			slice[i] = nodeSliceToSlice(val)
		default:
			slice[i] = val
		}
	}

	return slice
}

func (n *MapXNode) Data() interface{} {
	switch val := n.value.(type) {
	case map[string]*MapXNode:
		return nodeMsnToMsi(val)
	case []interface{}:
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

func (n *MapXNode) Msi() (map[string]interface{}, error) {
	var ok bool
	var msn map[string]*MapXNode

	if msn, ok = n.value.(map[string]*MapXNode); !ok {
		return nil, fmt.Errorf("value should be of type map[string]*MapXNode but instead is %T", n.value)
	}

	msi := nodeMsnToMsi(msn)

	return msi, nil
}

func (n *MapXNode) Slice() ([]interface{}, error) {
	var ok bool
	var slice []interface{}

	if slice, ok = n.value.([]interface{}); !ok {
		return nil, fmt.Errorf("value should be of type []interface{} but instead is %T", n.value)
	}

	return nodeSliceToSlice(slice), nil
}
