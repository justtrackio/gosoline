package stream

import "fmt"

type ModelMsg struct {
	CrudType string
	Version  int
	ModelId  string
	Body     string
}

func CreateModelMsg(raw *Message) (*ModelMsg, error) {
	crudType, ok := raw.Attributes["type"].(string)

	if !ok {
		return nil, fmt.Errorf("type is not a string: %v", raw.Attributes["type"])
	}

	if _, ok := raw.Attributes["version"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'version'")
	}

	versionFloat, ok := raw.Attributes["version"].(float64)

	if !ok {
		return nil, fmt.Errorf("version is not an int: %v", raw.Attributes["version"])
	}

	version := int(versionFloat)

	if _, ok := raw.Attributes["modelId"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'modelId'")
	}

	modelId, ok := raw.Attributes["modelId"].(string)

	if !ok {
		return nil, fmt.Errorf("modelId is not a string: %v", raw.Attributes["modelId"])
	}

	return &ModelMsg{
		CrudType: crudType,
		Version:  version,
		ModelId:  modelId,
		Body:     raw.Body,
	}, nil
}
