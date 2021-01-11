package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type KeyValues map[string]*dynamodb.AttributeValue

type keyBuilder struct {
	metadata   KeyAware
	hashValue  interface{}
	rangeValue interface{}
}

func (b *keyBuilder) withHash(hashValue interface{}) {
	b.hashValue = hashValue
}

func (b *keyBuilder) withRange(rangeValue interface{}) {
	b.rangeValue = rangeValue
}

func (b *keyBuilder) buildKey(item interface{}) (KeyValues, error) {
	if b.hashValue != nil || b.rangeValue != nil {
		return b.fromValues(b.hashValue, b.rangeValue)
	}

	return b.fromItem(item)
}

func (b *keyBuilder) fromItem(item interface{}) (KeyValues, error) {
	if item == nil {
		return nil, fmt.Errorf("can not build key attributes from nil Item")
	}

	key, err := dynamodbattribute.MarshalMap(item)

	if err != nil {
		return nil, fmt.Errorf("error marshalling the key attributes: %w", err)
	}

	for f := range key {
		if !b.metadata.IsKeyField(f) {
			delete(key, f)
		}
	}

	return key, nil
}

func (b *keyBuilder) fromValues(values ...interface{}) (KeyValues, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("there is no key values provided")
	}

	if len(values) > 2 {
		return nil, fmt.Errorf("the keys should have 1 element for hash only or 2 for hash and range but instead has %d elements with values %v", len(values), values)
	}

	attributeValueMap := make(map[string]interface{})

	if b.metadata.GetHashKey() == nil {
		return nil, fmt.Errorf("there is no hash key defined")
	}

	if values[0] == nil {
		return nil, fmt.Errorf("the provided value for the hash key [%s] can not be nil", *b.metadata.GetHashKey())
	}

	attributeValueMap[*b.metadata.GetHashKey()] = values[0]

	if (b.metadata.GetRangeKey() == nil && len(values) == 1) || (b.metadata.GetRangeKey() == nil && values[1] == nil) {
		return b.marshal(attributeValueMap)
	}

	if b.metadata.GetRangeKey() != nil && (len(values) < 2 || values[1] == nil) {
		return nil, fmt.Errorf("you have to provide a value for the range key named '%s'", *b.metadata.GetRangeKey())
	}

	if b.metadata.GetRangeKey() == nil && values[1] != nil {
		return nil, fmt.Errorf("you are querying by range key value [%v] but the table has no range key defined", values[1])
	}

	attributeValueMap[*b.metadata.GetRangeKey()] = values[1]

	return b.marshal(attributeValueMap)
}

func (b *keyBuilder) marshal(attributeValueMap map[string]interface{}) (KeyValues, error) {
	attributeValues, err := dynamodbattribute.MarshalMap(attributeValueMap)

	if err != nil {
		return nil, fmt.Errorf("can not marshal keys: %w", err)
	}

	return attributeValues, nil
}
