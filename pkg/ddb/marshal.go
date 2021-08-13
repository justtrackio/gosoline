package ddb

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func NewEncoder() *attributevalue.Encoder {
	return attributevalue.NewEncoder(func(options *attributevalue.EncoderOptions) {
		options.TagKey = "json"
	})
}

func MarshalMap(in interface{}) (map[string]types.AttributeValue, error) {
	av, err := NewEncoder().Encode(in)

	asMap, ok := av.(*types.AttributeValueMemberM)
	if err != nil || av == nil || !ok {
		return map[string]types.AttributeValue{}, err
	}

	return asMap.Value, nil
}
