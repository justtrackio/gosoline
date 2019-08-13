package ddb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"reflect"
)

type Unmarshaller struct {
	result reflect.Value
	typ    reflect.Type
}

func NewUnmarshallerFromPtrSlice(result interface{}) (*Unmarshaller, error) {
	ptr := reflect.ValueOf(result)

	if ptr.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("result interface is not a pointer")
	}

	value := ptr.Elem()

	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("result interface is not a slice")
	}

	um := &Unmarshaller{
		result: value,
		typ:    value.Type(),
	}

	return um, nil
}

func NewUnmarshallerFromStruct(model interface{}) (*Unmarshaller, error) {
	elemType := reflect.TypeOf(model)
	elemSlice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 10)

	um := &Unmarshaller{
		result: elemSlice,
		typ:    elemSlice.Type(),
	}

	return um, nil
}

func (u *Unmarshaller) Unmarshal(items []map[string]*dynamodb.AttributeValue) (interface{}, error) {
	partValue := reflect.New(u.typ)
	part := partValue.Interface()

	err := dynamodbattribute.UnmarshalListOfMaps(items, &part)

	if err != nil {
		return nil, err
	}

	indirect := reflect.Indirect(partValue)
	result := indirect.Interface()

	return result, nil
}

func (u *Unmarshaller) Append(items []map[string]*dynamodb.AttributeValue) error {
	partValue := reflect.New(u.typ)
	part := partValue.Interface()

	indirect := reflect.Indirect(partValue)

	err := dynamodbattribute.UnmarshalListOfMaps(items, &part)

	if err != nil {
		return err
	}

	for i := 0; i < indirect.Len(); i++ {
		u.result.Set(reflect.Append(u.result, indirect.Index(i)))
	}

	return nil
}
