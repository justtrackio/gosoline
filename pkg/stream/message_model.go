package stream

import (
	"fmt"
	"strconv"
)

type ModelMsg struct {
	CrudType string
	Version  int
	ModelId  string
	Body     string
}

func CreateModelMsg(raw *Message) (*ModelMsg, error) {
	crudType, ok := raw.Attributes["type"]

	if !ok {
		return nil, fmt.Errorf("type is not a string: %v", raw.Attributes["type"])
	}

	if _, ok := raw.Attributes["version"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'version'")
	}

	version, err := strconv.Atoi(raw.Attributes["version"])
	if err != nil {
		return nil, fmt.Errorf("version %q can not be parsed to an int: %w", raw.Attributes["version"], err)
	}

	if _, ok := raw.Attributes["modelId"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'modelId'")
	}

	modelId, ok := raw.Attributes["modelId"]

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
