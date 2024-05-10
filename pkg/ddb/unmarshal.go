package ddb

import (
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Unmarshaller struct {
	result reflect.Value
	typ    reflect.Type
}

func NewUnmarshallerFromPtrSlice(result any) (*Unmarshaller, error) {
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

func NewUnmarshallerFromStruct(model any) (*Unmarshaller, error) {
	elemType := reflect.TypeOf(model)
	elemSlice := reflect.MakeSlice(reflect.SliceOf(elemType), 0, 10)

	um := &Unmarshaller{
		result: elemSlice,
		typ:    elemSlice.Type(),
	}

	return um, nil
}

func (u *Unmarshaller) Unmarshal(items []map[string]types.AttributeValue) (any, error) {
	partValue := reflect.New(u.typ)
	part := partValue.Interface()

	err := UnmarshalListOfMaps(items, &part)
	if err != nil {
		return nil, err
	}

	indirect := reflect.Indirect(partValue)
	result := indirect.Interface()

	return result, nil
}

func (u *Unmarshaller) Append(items []map[string]types.AttributeValue) error {
	partValue := reflect.New(u.typ)
	part := partValue.Interface()

	indirect := reflect.Indirect(partValue)

	err := UnmarshalListOfMaps(items, &part)
	if err != nil {
		return err
	}

	for i := 0; i < indirect.Len(); i++ {
		u.result.Set(reflect.Append(u.result, indirect.Index(i)))
	}

	return nil
}

func NewDecoder() *attributevalue.Decoder {
	return attributevalue.NewDecoder(func(options *attributevalue.DecoderOptions) {
		options.TagKey = "json"
	})
}

func UnmarshalListOfMaps(l []map[string]types.AttributeValue, out any) error {
	items := make([]types.AttributeValue, len(l))
	for i, m := range l {
		items[i] = &types.AttributeValueMemberM{Value: m}
	}

	return UnmarshalList(items, out)
}

func UnmarshalList(l []types.AttributeValue, out any) error {
	return NewDecoder().Decode(&types.AttributeValueMemberL{Value: l}, out)
}

func UnmarshalMap(m map[string]types.AttributeValue, out any) error {
	return NewDecoder().Decode(&types.AttributeValueMemberM{Value: m}, out)
}
