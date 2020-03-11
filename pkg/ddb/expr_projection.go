package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

func buildProjectionExpression(metadata FieldAware, model interface{}) (*expression.ProjectionBuilder, error) {
	if model == nil {
		return nil, nil
	}

	projectedFields, err := MetadataReadFields(model)

	if err != nil {
		return nil, err
	}

	for _, f := range projectedFields {
		if !metadata.ContainsField(f) {
			return nil, fmt.Errorf("model of type %T has unknown fields: %s", model, f)
		}
	}

	if len(projectedFields) == len(metadata.GetFields()) {
		return nil, nil
	}

	projection := expression.ProjectionBuilder{}

	for _, f := range projectedFields {
		projection = expression.AddNames(projection, expression.Name(f))
	}

	return &projection, nil
}

func resolveTargetType(metadata FieldAware, projection interface{}, result interface{}) interface{} {
	if projection != nil {
		return projection
	}

	if _, ok := isResultCallback(result); ok {
		return metadata.GetModel()
	}

	if result != nil {
		return result
	}

	return metadata.GetModel()
}
